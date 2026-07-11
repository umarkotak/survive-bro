package room

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"survive-bro/apps/backend/internal/observability"
	"survive-bro/apps/backend/internal/protocol"
	"survive-bro/apps/backend/internal/simulation"
)

const MaxPlayers = 6

var (
	ErrClosed                  = errors.New("room closed")
	ErrFull                    = errors.New("room full")
	ErrInvalidDisplayName      = errors.New("display name must contain 1 to 20 characters")
	ErrReconnectNotImplemented = errors.New("reconnection is not implemented yet")
)

type State string

const (
	StateLobby    State = "lobby"
	StateRunning  State = "running"
	StateFinished State = "finished"
)

type Summary struct {
	Code        string
	State       State
	PlayerCount int
	ExpiresAt   time.Time
	LevelID     string
}

type JoinResult struct {
	PlayerID       string
	ReconnectToken string
	Host           bool
}

type Player struct {
	ID             string
	DisplayName    string
	ReconnectToken string
	JoinedAt       time.Time
	Send           chan protocol.Envelope
}

type Room struct {
	id       string
	code     string
	ttl      time.Duration
	now      func() time.Time
	commands chan any
	done     chan struct{}
	onClose  func(string, *Room)
	metrics  *observability.Collector
	level    simulation.LevelDefinition
}

type summaryCommand struct{ response chan Summary }

type joinCommand struct {
	displayName    string
	reconnectToken *string
	send           chan protocol.Envelope
	response       chan joinResponse
}

type joinResponse struct {
	result JoinResult
	err    error
}

type messageCommand struct {
	playerID string
	message  protocol.Envelope
}

type outboundCommand struct {
	playerID string
	message  protocol.Envelope
}

type leaveCommand struct {
	playerID string
	done     chan struct{}
}

type closeCommand struct{ done chan struct{} }

func newRoom(id, code string, level simulation.LevelDefinition, ttl time.Duration, now func() time.Time, metrics *observability.Collector, onClose func(string, *Room)) *Room {
	room := &Room{
		id:       id,
		code:     code,
		ttl:      ttl,
		now:      now,
		commands: make(chan any, 64),
		done:     make(chan struct{}),
		onClose:  onClose,
		metrics:  metrics,
		level:    level,
	}
	go room.run()
	return room
}

func (r *Room) Code() string { return r.code }

func (r *Room) Summary(ctx context.Context) (Summary, error) {
	response := make(chan Summary, 1)
	if err := r.send(ctx, summaryCommand{response: response}); err != nil {
		return Summary{}, err
	}
	select {
	case summary := <-response:
		return summary, nil
	case <-ctx.Done():
		return Summary{}, ctx.Err()
	case <-r.done:
		return Summary{}, ErrClosed
	}
}

func (r *Room) Join(ctx context.Context, displayName string, reconnectToken *string, send chan protocol.Envelope) (JoinResult, error) {
	response := make(chan joinResponse, 1)
	command := joinCommand{displayName: displayName, reconnectToken: reconnectToken, send: send, response: response}
	if err := r.send(ctx, command); err != nil {
		return JoinResult{}, err
	}
	select {
	case result := <-response:
		return result.result, result.err
	case <-ctx.Done():
		return JoinResult{}, ctx.Err()
	case <-r.done:
		return JoinResult{}, ErrClosed
	}
}

func (r *Room) Handle(ctx context.Context, playerID string, message protocol.Envelope) error {
	return r.send(ctx, messageCommand{playerID: playerID, message: message})
}

func (r *Room) Send(ctx context.Context, playerID string, message protocol.Envelope) error {
	return r.send(ctx, outboundCommand{playerID: playerID, message: message})
}

func (r *Room) Leave(ctx context.Context, playerID string) error {
	done := make(chan struct{})
	if err := r.send(ctx, leaveCommand{playerID: playerID, done: done}); err != nil {
		return err
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-r.done:
		return nil
	}
}

func (r *Room) Close(ctx context.Context) error {
	done := make(chan struct{})
	if err := r.send(ctx, closeCommand{done: done}); err != nil {
		if errors.Is(err, ErrClosed) {
			return nil
		}
		return err
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-r.done:
		return nil
	}
}

func (r *Room) send(ctx context.Context, command any) error {
	select {
	case r.commands <- command:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-r.done:
		return ErrClosed
	}
}

func (r *Room) run() {
	defer close(r.done)
	defer r.onClose(r.code, r)

	clients := make(map[string]*Player)
	hostPlayerID := ""
	var match *simulation.Match
	expiresAt := r.now().Add(r.ttl)
	expiryTimer := time.NewTimer(time.Until(expiresAt))
	ticker := time.NewTicker(simulation.TickDuration)
	defer expiryTimer.Stop()
	defer ticker.Stop()

	for {
		select {
		case raw := <-r.commands:
			switch command := raw.(type) {
			case summaryCommand:
				command.response <- Summary{Code: r.code, State: matchState(match), PlayerCount: len(clients), ExpiresAt: expiresAt, LevelID: r.level.ID}
			case joinCommand:
				result, createdMatch, err := r.join(clients, &hostPlayerID, &match, command)
				command.response <- joinResponse{result: result, err: err}
				if err != nil {
					continue
				}
				stopTimer(expiryTimer)
				if createdMatch {
					r.broadcastMatchStarted(clients, match)
				} else {
					r.sendMatchStarted(clients[result.PlayerID], match)
				}
				r.broadcastAndPrune(clients, &hostPlayerID, match)
				r.broadcastSnapshot(clients, match)
				if len(clients) == 0 {
					match = nil
					expiresAt = r.now().Add(r.ttl)
					resetTimer(expiryTimer, time.Until(expiresAt))
				}
			case messageCommand:
				if slowPlayerID := r.handleMessage(clients, match, command); slowPlayerID != "" {
					r.removePlayer(clients, &hostPlayerID, match, slowPlayerID)
					r.broadcastAndPrune(clients, &hostPlayerID, match)
				}
				if len(clients) == 0 {
					match = nil
					expiresAt = r.now().Add(r.ttl)
					resetTimer(expiryTimer, time.Until(expiresAt))
				}
			case outboundCommand:
				if !r.sendCritical(clients, command.playerID, command.message) {
					r.removePlayer(clients, &hostPlayerID, match, command.playerID)
					r.broadcastAndPrune(clients, &hostPlayerID, match)
				}
			case leaveCommand:
				if r.removePlayer(clients, &hostPlayerID, match, command.playerID) {
					r.broadcastAndPrune(clients, &hostPlayerID, match)
				}
				if len(clients) == 0 {
					match = nil
					expiresAt = r.now().Add(r.ttl)
					resetTimer(expiryTimer, time.Until(expiresAt))
				}
				close(command.done)
			case closeCommand:
				r.shutdownPlayers(clients)
				close(command.done)
				return
			}
		case now := <-ticker.C:
			if match == nil || len(clients) == 0 {
				continue
			}
			events := match.Step(now)
			r.metrics.RecordTick(observability.TickSample{
				Room: r.code, Total: events.Metrics.Total, Movement: events.Metrics.Movement, Weapons: events.Metrics.Weapons,
				ProjectileMove: events.Metrics.ProjectileMove, BroadPhase: events.Metrics.BroadPhase, NarrowPhase: events.Metrics.NarrowPhase,
				EnemyAI: events.Metrics.EnemyAI, Pickups: events.Metrics.Pickups, Spawning: events.Metrics.Spawning,
				Players: len(match.Players), Monsters: len(match.Monsters), Projectiles: len(match.Projectiles), PickupsCount: len(match.Pickups),
				CandidatePairs: events.Metrics.CandidatePairs, NarrowChecks: events.Metrics.NarrowChecks, ConfirmedHits: events.Metrics.ConfirmedHits,
			})
			removedSlowClient := false
			for _, playerID := range r.broadcastSimulationEvents(clients, events) {
				removedSlowClient = r.removePlayer(clients, &hostPlayerID, match, playerID) || removedSlowClient
			}
			if removedSlowClient {
				r.broadcastAndPrune(clients, &hostPlayerID, match)
			}
			if match.Tick%simulation.SnapshotEveryTicks == 0 || events.MatchEnded != nil {
				r.broadcastSnapshot(clients, match)
			}
			if events.MatchEnded != nil {
				r.broadcastAndPrune(clients, &hostPlayerID, match)
			}
			r.recordQueueDepth(clients)
			if len(clients) == 0 {
				match = nil
				expiresAt = r.now().Add(r.ttl)
				resetTimer(expiryTimer, time.Until(expiresAt))
			}
		case <-expiryTimer.C:
			if len(clients) == 0 {
				r.shutdownPlayers(clients)
				return
			}
		}
	}
}

func (r *Room) join(clients map[string]*Player, hostPlayerID *string, match **simulation.Match, command joinCommand) (JoinResult, bool, error) {
	name := strings.TrimSpace(command.displayName)
	if count := utf8.RuneCountInString(name); count < 1 || count > 20 {
		return JoinResult{}, false, ErrInvalidDisplayName
	}
	if command.reconnectToken != nil {
		return JoinResult{}, false, ErrReconnectNotImplemented
	}
	if len(clients) >= MaxPlayers {
		return JoinResult{}, false, ErrFull
	}

	playerID, err := secureToken("p_", 12)
	if err != nil {
		return JoinResult{}, false, err
	}
	reconnectToken, err := secureToken("", 32)
	if err != nil {
		return JoinResult{}, false, err
	}
	client := &Player{ID: playerID, DisplayName: name, ReconnectToken: reconnectToken, JoinedAt: r.now(), Send: command.send}
	clients[playerID] = client
	if *hostPlayerID == "" {
		*hostPlayerID = playerID
	}

	createdMatch := *match == nil || (*match).Finished
	if createdMatch {
		*match = simulation.NewMatch(r.now(), r.now().UnixNano(), r.level)
		for _, existing := range orderedPlayers(clients) {
			(*match).AddPlayer(existing.ID, existing.DisplayName, r.now())
		}
	} else {
		(*match).AddPlayer(playerID, name, r.now())
	}

	joined, err := protocol.NewEnvelope(protocol.TypeJoined, "", protocol.JoinedPayload{
		PlayerID:       playerID,
		ReconnectToken: reconnectToken,
		RoomName:       r.code,
		Host:           *hostPlayerID == playerID,
	})
	if err != nil {
		delete(clients, playerID)
		(*match).RemovePlayer(playerID)
		return JoinResult{}, false, err
	}
	command.send <- joined
	return JoinResult{PlayerID: playerID, ReconnectToken: reconnectToken, Host: *hostPlayerID == playerID}, createdMatch, nil
}

func (r *Room) handleMessage(clients map[string]*Player, match *simulation.Match, command messageCommand) string {
	client, ok := clients[command.playerID]
	if !ok {
		return ""
	}
	switch command.message.Type {
	case protocol.TypePing:
		pong, err := protocol.NewEnvelope(protocol.TypePong, command.message.RequestID, struct{}{})
		if err == nil && !r.sendCritical(clients, client.ID, pong) {
			return client.ID
		}
	case protocol.TypeInput:
		if match == nil || match.Finished {
			return ""
		}
		var payload protocol.InputPayload
		if err := command.message.DecodePayload(&payload); err != nil {
			r.sendCritical(clients, client.ID, protocol.Error(command.message.RequestID, "invalid_payload", "input payload is invalid"))
			return ""
		}
		if err := match.ApplyInput(client.ID, payload, r.now()); err != nil {
			r.sendCritical(clients, client.ID, protocol.Error(command.message.RequestID, "invalid_input", "movement axes must be finite values between -1 and 1"))
		}
	case protocol.TypeLeaveRoom:
		// Transport performs synchronized removal after returning from its read loop.
	default:
		if !r.sendCritical(clients, client.ID, protocol.Error(command.message.RequestID, "unsupported_message", "message is not available in the current multiplayer milestone")) {
			return client.ID
		}
	}
	return ""
}

func (r *Room) sendMatchStarted(client *Player, match *simulation.Match) {
	if client == nil || match == nil {
		return
	}
	payload := protocol.MatchStartedPayload{
		RoomName:    r.code,
		MapID:       r.level.ID,
		MapWidth:    simulation.WorldWidth,
		MapHeight:   simulation.WorldHeight,
		StartedAtMs: match.StartedAt.UnixMilli(),
		Obstacles:   make([]protocol.Obstacle, 0, len(r.level.Obstacles)),
	}
	for _, obstacle := range r.level.Obstacles {
		payload.Obstacles = append(payload.Obstacles, protocol.Obstacle{ID: obstacle.ID, Type: obstacle.Type, X: obstacle.X, Y: obstacle.Y, Radius: obstacle.Radius})
	}
	envelope, err := protocol.NewEnvelope(protocol.TypeMatchStarted, "", payload)
	if err == nil {
		r.sendCritical(map[string]*Player{client.ID: client}, client.ID, envelope)
	}
}

func (r *Room) broadcastMatchStarted(clients map[string]*Player, match *simulation.Match) {
	for _, client := range clients {
		r.sendMatchStarted(client, match)
	}
}

func (r *Room) broadcastSimulationEvents(clients map[string]*Player, events simulation.Events) []string {
	slow := make(map[string]struct{})
	for _, spawned := range events.SpawnedProjectiles {
		for _, playerID := range r.broadcastCritical(clients, protocol.TypeProjectileSpawned, spawned) {
			slow[playerID] = struct{}{}
		}
	}
	for _, removed := range events.RemovedProjectiles {
		for _, playerID := range r.broadcastCritical(clients, protocol.TypeProjectileRemoved, protocol.ProjectileRemovedPayload{ProjectileID: removed.ID, Reason: removed.Reason}) {
			slow[playerID] = struct{}{}
		}
	}
	for _, upgrade := range events.AppliedUpgrades {
		for _, playerID := range r.broadcastCritical(clients, protocol.TypeUpgradeApplied, upgrade) {
			slow[playerID] = struct{}{}
		}
	}
	if events.MatchEnded != nil {
		for _, playerID := range r.broadcastCritical(clients, protocol.TypeMatchEnded, *events.MatchEnded) {
			slow[playerID] = struct{}{}
		}
	}
	playerIDs := make([]string, 0, len(slow))
	for playerID := range slow {
		playerIDs = append(playerIDs, playerID)
	}
	return playerIDs
}

func (r *Room) broadcastCritical(clients map[string]*Player, messageType protocol.MessageType, payload any) []string {
	envelope, err := protocol.NewEnvelope(messageType, "", payload)
	if err != nil {
		return nil
	}
	slow := make([]string, 0)
	for playerID := range clients {
		if !r.sendCritical(clients, playerID, envelope) {
			slow = append(slow, playerID)
		}
	}
	return slow
}

func (r *Room) broadcastSnapshot(clients map[string]*Player, match *simulation.Match) {
	if match == nil {
		return
	}
	snapshotStarted := time.Now()
	snapshot := match.Snapshot(r.now())
	r.metrics.RecordSnapshotBuild(time.Since(snapshotStarted))
	envelope, err := protocol.NewEnvelope(protocol.TypeSnapshot, "", snapshot)
	if err != nil {
		return
	}
	enqueueStarted := time.Now()
	defer func() { r.metrics.RecordWebSocketEnqueue(time.Since(enqueueStarted)) }()
	for _, client := range clients {
		select {
		case client.Send <- envelope:
		default:
			r.metrics.RecordSnapshotReplaced()
			// Snapshots are replaceable; critical events will disconnect a persistently slow client.
		}
	}
}

func (r *Room) broadcastRoomState(clients map[string]*Player, hostPlayerID string, state State) []string {
	states := make([]protocol.PlayerState, 0, len(clients))
	for _, player := range orderedPlayers(clients) {
		states = append(states, protocol.PlayerState{
			ID:          player.ID,
			DisplayName: player.DisplayName,
			CharacterID: "ranger",
			Ready:       true,
			Connected:   true,
		})
	}
	envelope, err := protocol.NewEnvelope(protocol.TypeRoomState, "", protocol.RoomStatePayload{Status: string(state), HostPlayerID: hostPlayerID, Players: states})
	if err != nil {
		return nil
	}
	slowPlayerIDs := make([]string, 0)
	for playerID := range clients {
		if !r.sendCritical(clients, playerID, envelope) {
			slowPlayerIDs = append(slowPlayerIDs, playerID)
		}
	}
	return slowPlayerIDs
}

func (r *Room) sendCritical(clients map[string]*Player, playerID string, envelope protocol.Envelope) bool {
	started := time.Now()
	defer func() { r.metrics.RecordWebSocketEnqueue(time.Since(started)) }()
	player, ok := clients[playerID]
	if !ok {
		return true
	}
	select {
	case player.Send <- envelope:
		return true
	default:
		r.metrics.RecordCriticalQueueFailure()
		return false
	}
}

func (r *Room) recordQueueDepth(clients map[string]*Player) {
	total := 0
	maximum := 0
	for _, player := range clients {
		depth := len(player.Send)
		total += depth
		maximum = max(maximum, depth)
	}
	r.metrics.RecordRoomQueueDepth(r.code, total, maximum)
}

func (r *Room) broadcastAndPrune(clients map[string]*Player, hostPlayerID *string, match *simulation.Match) {
	for {
		slowPlayerIDs := r.broadcastRoomState(clients, *hostPlayerID, matchState(match))
		if len(slowPlayerIDs) == 0 {
			return
		}
		for _, playerID := range slowPlayerIDs {
			r.removePlayer(clients, hostPlayerID, match, playerID)
		}
	}
}

func (r *Room) removePlayer(clients map[string]*Player, hostPlayerID *string, match *simulation.Match, playerID string) bool {
	player, ok := clients[playerID]
	if !ok {
		return false
	}
	delete(clients, playerID)
	close(player.Send)
	if match != nil {
		match.RemovePlayer(playerID)
	}
	if *hostPlayerID == playerID {
		*hostPlayerID = oldestPlayerID(clients)
	}
	return true
}

func (r *Room) shutdownPlayers(clients map[string]*Player) {
	message, _ := protocol.NewEnvelope(protocol.TypeServerClosed, "", protocol.ServerShutdownPayload{Reason: "server shutting down"})
	for playerID, player := range clients {
		select {
		case player.Send <- message:
		default:
		}
		close(player.Send)
		delete(clients, playerID)
	}
}

func matchState(match *simulation.Match) State {
	if match == nil {
		return StateLobby
	}
	if match.Finished {
		return StateFinished
	}
	return StateRunning
}

func orderedPlayers(players map[string]*Player) []*Player {
	ordered := make([]*Player, 0, len(players))
	for _, player := range players {
		ordered = append(ordered, player)
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].JoinedAt.Equal(ordered[j].JoinedAt) {
			return ordered[i].ID < ordered[j].ID
		}
		return ordered[i].JoinedAt.Before(ordered[j].JoinedAt)
	})
	return ordered
}

func oldestPlayerID(players map[string]*Player) string {
	ordered := orderedPlayers(players)
	if len(ordered) == 0 {
		return ""
	}
	return ordered[0].ID
}

func secureToken(prefix string, size int) (string, error) {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return prefix + base64.RawURLEncoding.EncodeToString(buffer), nil
}

func stopTimer(timer *time.Timer) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
}

func resetTimer(timer *time.Timer, duration time.Duration) {
	stopTimer(timer)
	timer.Reset(duration)
}
