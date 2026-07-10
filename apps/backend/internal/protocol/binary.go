package protocol

import (
	"encoding/binary"
	"fmt"
	"math"
	"unicode/utf8"
)

const (
	maxDecodedPlayers   = 4
	maxDecodedObstacles = 256
	maxDecodedMonsters  = 1024
	maxDecodedPickups   = 2048
)

func Encode(envelope Envelope) ([]byte, error) {
	if envelope.Version != Version {
		return nil, fmt.Errorf("protocol version must be %d", Version)
	}
	if !envelope.Type.valid() {
		return nil, fmt.Errorf("unknown message type %d", envelope.Type)
	}
	if len(envelope.RequestID) > math.MaxUint8 || !utf8.ValidString(envelope.RequestID) {
		return nil, fmt.Errorf("request ID must be valid UTF-8 and at most 255 bytes")
	}
	encoder := binaryEncoder{data: make([]byte, 0, initialCapacity(envelope))}
	encoder.u8(Version)
	encoder.u8(uint8(envelope.Type))
	encoder.u8(uint8(len(envelope.RequestID)))
	encoder.data = append(encoder.data, envelope.RequestID...)
	if err := encoder.payload(envelope.Type, envelope.Payload); err != nil {
		return nil, err
	}
	return encoder.data, nil
}

func Decode(data []byte) (Envelope, error) {
	decoder := binaryDecoder{data: data}
	version, err := decoder.u8()
	if err != nil {
		return Envelope{}, err
	}
	if version != Version {
		return Envelope{}, fmt.Errorf("unsupported protocol version %d", version)
	}
	typeID, err := decoder.u8()
	if err != nil {
		return Envelope{}, err
	}
	messageType := MessageType(typeID)
	if !messageType.valid() {
		return Envelope{}, fmt.Errorf("unknown message type %d", typeID)
	}
	requestLength, err := decoder.u8()
	if err != nil {
		return Envelope{}, err
	}
	requestBytes, err := decoder.bytes(int(requestLength))
	if err != nil {
		return Envelope{}, err
	}
	if !utf8.Valid(requestBytes) {
		return Envelope{}, fmt.Errorf("request ID is not valid UTF-8")
	}
	payload, err := decoder.payload(messageType)
	if err != nil {
		return Envelope{}, err
	}
	if decoder.remaining() != 0 {
		return Envelope{}, fmt.Errorf("frame contains %d trailing bytes", decoder.remaining())
	}
	return Envelope{Version: version, Type: messageType, RequestID: string(requestBytes), Payload: payload}, nil
}

type binaryEncoder struct{ data []byte }

func (e *binaryEncoder) u8(value uint8) { e.data = append(e.data, value) }

func (e *binaryEncoder) u16(value uint16) {
	start := len(e.data)
	e.data = append(e.data, 0, 0)
	binary.LittleEndian.PutUint16(e.data[start:], value)
}

func (e *binaryEncoder) u32(value uint32) {
	start := len(e.data)
	e.data = append(e.data, 0, 0, 0, 0)
	binary.LittleEndian.PutUint32(e.data[start:], value)
}

func (e *binaryEncoder) i64(value int64) {
	start := len(e.data)
	e.data = append(e.data, 0, 0, 0, 0, 0, 0, 0, 0)
	binary.LittleEndian.PutUint64(e.data[start:], uint64(value))
}

func initialCapacity(envelope Envelope) int {
	switch envelope.Type {
	case TypeSnapshot:
		if snapshot, ok := envelope.Payload.(SnapshotPayload); ok {
			return 64 + len(snapshot.Players)*80 + len(snapshot.Monsters)*18 + len(snapshot.Pickups)*14
		}
	case TypeMatchStarted:
		if started, ok := envelope.Payload.(MatchStartedPayload); ok {
			return 64 + len(started.Obstacles)*48
		}
	case TypeRoomState:
		return 384
	case TypeError, TypeServerClosed:
		return 256
	}
	return 128
}

func (e *binaryEncoder) f32(value float64) error {
	if math.IsNaN(value) || math.IsInf(value, 0) || value > math.MaxFloat32 || value < -math.MaxFloat32 {
		return fmt.Errorf("float value %v is not finite float32", value)
	}
	e.u32(math.Float32bits(float32(value)))
	return nil
}

func (e *binaryEncoder) string(value string) error {
	if !utf8.ValidString(value) || len(value) > math.MaxUint16 {
		return fmt.Errorf("string must be valid UTF-8 and at most 65535 bytes")
	}
	e.u16(uint16(len(value)))
	e.data = append(e.data, value...)
	return nil
}

func (e *binaryEncoder) bool(value bool) {
	if value {
		e.u8(1)
	} else {
		e.u8(0)
	}
}

func (e *binaryEncoder) payload(messageType MessageType, payload any) error {
	switch messageType {
	case TypeJoinRoom:
		value, ok := payload.(JoinRoomPayload)
		if !ok {
			return payloadTypeError(messageType, payload)
		}
		if err := e.string(value.DisplayName); err != nil {
			return err
		}
		e.bool(value.ReconnectToken != nil)
		if value.ReconnectToken != nil {
			return e.string(*value.ReconnectToken)
		}
	case TypeLeaveRoom, TypePing, TypePong:
		if payload != nil {
			switch payload.(type) {
			case struct{}:
			default:
				return payloadTypeError(messageType, payload)
			}
		}
	case TypeInput:
		value, ok := payload.(InputPayload)
		if !ok {
			return payloadTypeError(messageType, payload)
		}
		sequence, err := uint32Value(value.Sequence, "input sequence")
		if err != nil {
			return err
		}
		e.u32(sequence)
		if err := e.f32(value.MoveX); err != nil {
			return err
		}
		return e.f32(value.MoveY)
	case TypeJoined:
		value, ok := payload.(JoinedPayload)
		if !ok {
			return payloadTypeError(messageType, payload)
		}
		for _, field := range []string{value.PlayerID, value.ReconnectToken, value.RoomName} {
			if err := e.string(field); err != nil {
				return err
			}
		}
		e.bool(value.Host)
	case TypeRoomState:
		value, ok := payload.(RoomStatePayload)
		if !ok {
			return payloadTypeError(messageType, payload)
		}
		status, err := encodeStatus(value.Status)
		if err != nil {
			return err
		}
		e.u8(status)
		if err := e.string(value.HostPlayerID); err != nil {
			return err
		}
		if len(value.Players) > math.MaxUint8 {
			return fmt.Errorf("player count exceeds 255")
		}
		e.u8(uint8(len(value.Players)))
		for _, player := range value.Players {
			for _, field := range []string{player.ID, player.DisplayName, player.CharacterID} {
				if err := e.string(field); err != nil {
					return err
				}
			}
			var flags uint8
			if player.Ready {
				flags |= 1
			}
			if player.Connected {
				flags |= 2
			}
			e.u8(flags)
		}
	case TypeMatchStarted:
		value, ok := payload.(MatchStartedPayload)
		if !ok {
			return payloadTypeError(messageType, payload)
		}
		if err := e.string(value.RoomName); err != nil {
			return err
		}
		if err := e.string(value.MapID); err != nil {
			return err
		}
		if err := e.f32(value.MapWidth); err != nil {
			return err
		}
		if err := e.f32(value.MapHeight); err != nil {
			return err
		}
		e.i64(value.StartedAtMs)
		if len(value.Obstacles) > math.MaxUint16 {
			return fmt.Errorf("obstacle count exceeds 65535")
		}
		e.u16(uint16(len(value.Obstacles)))
		for _, obstacle := range value.Obstacles {
			if err := e.string(obstacle.ID); err != nil {
				return err
			}
			if err := e.string(obstacle.Type); err != nil {
				return err
			}
			for _, field := range []float64{obstacle.X, obstacle.Y, obstacle.Radius} {
				if err := e.f32(field); err != nil {
					return err
				}
			}
		}
	case TypeSnapshot:
		value, ok := payload.(SnapshotPayload)
		if !ok {
			return payloadTypeError(messageType, payload)
		}
		return e.snapshot(value)
	case TypeProjectileSpawned:
		value, ok := payload.(ProjectileSpawnedPayload)
		if !ok {
			return payloadTypeError(messageType, payload)
		}
		projectileID, err := uint32Value(value.ProjectileID, "projectile ID")
		if err != nil {
			return err
		}
		e.u32(projectileID)
		if err := e.string(value.OwnerID); err != nil {
			return err
		}
		if err := e.string(value.WeaponID); err != nil {
			return err
		}
		for _, field := range []float64{value.X, value.Y, value.VelocityX, value.VelocityY} {
			if err := e.f32(field); err != nil {
				return err
			}
		}
		spawnTick, err := uint32Value(value.SpawnTick, "spawn tick")
		if err != nil {
			return err
		}
		e.u32(spawnTick)
	case TypeProjectileRemoved:
		value, ok := payload.(ProjectileRemovedPayload)
		if !ok {
			return payloadTypeError(messageType, payload)
		}
		projectileID, err := uint32Value(value.ProjectileID, "projectile ID")
		if err != nil {
			return err
		}
		e.u32(projectileID)
		reason, err := encodeRemovalReason(value.Reason)
		if err != nil {
			return err
		}
		e.u8(reason)
	case TypeMatchEnded:
		value, ok := payload.(MatchEndedPayload)
		if !ok {
			return payloadTypeError(messageType, payload)
		}
		outcome, err := encodeOutcome(value.Outcome)
		if err != nil {
			return err
		}
		e.u8(outcome)
		survival, err := nonNegativeUint32(value.SurvivalMs, "survival ms")
		if err != nil {
			return err
		}
		e.u32(survival)
		level, err := nonNegativeUint16(value.TeamLevel, "team level")
		if err != nil {
			return err
		}
		e.u16(level)
		kills, err := nonNegativeUint32Int(value.TotalKills, "total kills")
		if err != nil {
			return err
		}
		e.u32(kills)
	case TypeError:
		value, ok := payload.(ErrorPayload)
		if !ok {
			return payloadTypeError(messageType, payload)
		}
		if err := e.string(value.Code); err != nil {
			return err
		}
		return e.string(value.Message)
	case TypeServerClosed:
		value, ok := payload.(ServerShutdownPayload)
		if !ok {
			return payloadTypeError(messageType, payload)
		}
		return e.string(value.Reason)
	}
	return nil
}

func (e *binaryEncoder) snapshot(value SnapshotPayload) error {
	tick, err := uint32Value(value.Tick, "snapshot tick")
	if err != nil {
		return err
	}
	e.u32(tick)
	e.i64(value.ServerTimeMs)
	if len(value.Players) > math.MaxUint8 {
		return fmt.Errorf("player count exceeds 255")
	}
	e.u8(uint8(len(value.Players)))
	for _, player := range value.Players {
		if err := e.string(player.ID); err != nil {
			return err
		}
		if err := e.string(player.DisplayName); err != nil {
			return err
		}
		for _, field := range []float64{player.X, player.Y, player.VelocityX, player.VelocityY} {
			if err := e.f32(field); err != nil {
				return err
			}
		}
		var flags uint8
		if player.Facing == "left" {
			flags |= 1
		} else if player.Facing != "right" {
			return fmt.Errorf("unknown facing %q", player.Facing)
		}
		if player.Alive {
			flags |= 2
		}
		e.u8(flags)
		hp, err := nonNegativeUint16(player.HP, "player hp")
		if err != nil {
			return err
		}
		maxHP, err := nonNegativeUint16(player.MaxHP, "player max hp")
		if err != nil {
			return err
		}
		e.u16(hp)
		e.u16(maxHP)
		lastInput, err := uint32Value(player.LastProcessedInput, "last processed input")
		if err != nil {
			return err
		}
		e.u32(lastInput)
		kills, err := nonNegativeUint32Int(player.Kills, "player kills")
		if err != nil {
			return err
		}
		e.u32(kills)
	}
	if len(value.Monsters) > math.MaxUint16 {
		return fmt.Errorf("monster count exceeds 65535")
	}
	e.u16(uint16(len(value.Monsters)))
	for _, monster := range value.Monsters {
		id, err := uint32Value(monster.ID, "monster ID")
		if err != nil {
			return err
		}
		e.u32(id)
		if err := e.f32(monster.X); err != nil {
			return err
		}
		if err := e.f32(monster.Y); err != nil {
			return err
		}
		hp, err := nonNegativeUint16(monster.HP, "monster hp")
		if err != nil {
			return err
		}
		maxHP, err := nonNegativeUint16(monster.MaxHP, "monster max hp")
		if err != nil {
			return err
		}
		e.u16(hp)
		e.u16(maxHP)
	}
	if len(value.Pickups) > math.MaxUint16 {
		return fmt.Errorf("pickup count exceeds 65535")
	}
	e.u16(uint16(len(value.Pickups)))
	for _, pickup := range value.Pickups {
		id, err := uint32Value(pickup.ID, "pickup ID")
		if err != nil {
			return err
		}
		e.u32(id)
		if err := e.f32(pickup.X); err != nil {
			return err
		}
		if err := e.f32(pickup.Y); err != nil {
			return err
		}
	}
	level, err := nonNegativeUint16(value.Team.Level, "team level")
	if err != nil {
		return err
	}
	experience, err := nonNegativeUint16(value.Team.Experience, "team experience")
	if err != nil {
		return err
	}
	required, err := nonNegativeUint16(value.Team.ExperienceRequired, "team experience required")
	if err != nil {
		return err
	}
	kills, err := nonNegativeUint32Int(value.Team.TotalKills, "team total kills")
	if err != nil {
		return err
	}
	remaining, err := nonNegativeUint32(value.RemainingMs, "remaining ms")
	if err != nil {
		return err
	}
	e.u16(level)
	e.u16(experience)
	e.u16(required)
	e.u32(kills)
	e.u32(remaining)
	return nil
}

type binaryDecoder struct {
	data   []byte
	offset int
}

func (d *binaryDecoder) remaining() int { return len(d.data) - d.offset }

func (d *binaryDecoder) bytes(length int) ([]byte, error) {
	if length < 0 || d.remaining() < length {
		return nil, fmt.Errorf("binary frame is truncated at byte %d", d.offset)
	}
	value := d.data[d.offset : d.offset+length]
	d.offset += length
	return value, nil
}

func (d *binaryDecoder) u8() (uint8, error) {
	value, err := d.bytes(1)
	if err != nil {
		return 0, err
	}
	return value[0], nil
}

func (d *binaryDecoder) u16() (uint16, error) {
	value, err := d.bytes(2)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint16(value), nil
}

func (d *binaryDecoder) u32() (uint32, error) {
	value, err := d.bytes(4)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(value), nil
}

func (d *binaryDecoder) i64() (int64, error) {
	value, err := d.bytes(8)
	if err != nil {
		return 0, err
	}
	return int64(binary.LittleEndian.Uint64(value)), nil
}

func (d *binaryDecoder) f32() (float64, error) {
	value, err := d.u32()
	if err != nil {
		return 0, err
	}
	decoded := float64(math.Float32frombits(value))
	if math.IsNaN(decoded) || math.IsInf(decoded, 0) {
		return 0, fmt.Errorf("binary frame contains non-finite float")
	}
	return decoded, nil
}

func (d *binaryDecoder) string() (string, error) {
	length, err := d.u16()
	if err != nil {
		return "", err
	}
	value, err := d.bytes(int(length))
	if err != nil {
		return "", err
	}
	if !utf8.Valid(value) {
		return "", fmt.Errorf("binary frame contains invalid UTF-8")
	}
	return string(value), nil
}

func (d *binaryDecoder) bool() (bool, error) {
	value, err := d.u8()
	if err != nil {
		return false, err
	}
	if value > 1 {
		return false, fmt.Errorf("invalid boolean value %d", value)
	}
	return value == 1, nil
}

func (d *binaryDecoder) payload(messageType MessageType) (any, error) {
	switch messageType {
	case TypeJoinRoom:
		displayName, err := d.string()
		if err != nil {
			return nil, err
		}
		hasToken, err := d.bool()
		if err != nil {
			return nil, err
		}
		var token *string
		if hasToken {
			value, err := d.string()
			if err != nil {
				return nil, err
			}
			token = &value
		}
		return JoinRoomPayload{DisplayName: displayName, ReconnectToken: token}, nil
	case TypeLeaveRoom, TypePing, TypePong:
		return struct{}{}, nil
	case TypeInput:
		sequence, err := d.u32()
		if err != nil {
			return nil, err
		}
		moveX, err := d.f32()
		if err != nil {
			return nil, err
		}
		moveY, err := d.f32()
		if err != nil {
			return nil, err
		}
		return InputPayload{Sequence: uint64(sequence), MoveX: moveX, MoveY: moveY}, nil
	case TypeJoined:
		playerID, err := d.string()
		if err != nil {
			return nil, err
		}
		token, err := d.string()
		if err != nil {
			return nil, err
		}
		roomName, err := d.string()
		if err != nil {
			return nil, err
		}
		host, err := d.bool()
		return JoinedPayload{PlayerID: playerID, ReconnectToken: token, RoomName: roomName, Host: host}, err
	case TypeRoomState:
		return d.roomState()
	case TypeMatchStarted:
		return d.matchStarted()
	case TypeSnapshot:
		return d.snapshot()
	case TypeProjectileSpawned:
		return d.projectileSpawned()
	case TypeProjectileRemoved:
		id, err := d.u32()
		if err != nil {
			return nil, err
		}
		reasonID, err := d.u8()
		if err != nil {
			return nil, err
		}
		reason, err := decodeRemovalReason(reasonID)
		return ProjectileRemovedPayload{ProjectileID: uint64(id), Reason: reason}, err
	case TypeMatchEnded:
		outcomeID, err := d.u8()
		if err != nil {
			return nil, err
		}
		outcome, err := decodeOutcome(outcomeID)
		if err != nil {
			return nil, err
		}
		survival, err := d.u32()
		if err != nil {
			return nil, err
		}
		level, err := d.u16()
		if err != nil {
			return nil, err
		}
		kills, err := d.u32()
		return MatchEndedPayload{Outcome: outcome, SurvivalMs: int64(survival), TeamLevel: int(level), TotalKills: int(kills)}, err
	case TypeError:
		code, err := d.string()
		if err != nil {
			return nil, err
		}
		message, err := d.string()
		return ErrorPayload{Code: code, Message: message}, err
	case TypeServerClosed:
		reason, err := d.string()
		return ServerShutdownPayload{Reason: reason}, err
	}
	return nil, fmt.Errorf("unknown message type %d", messageType)
}

func (d *binaryDecoder) roomState() (RoomStatePayload, error) {
	statusID, err := d.u8()
	if err != nil {
		return RoomStatePayload{}, err
	}
	status, err := decodeStatus(statusID)
	if err != nil {
		return RoomStatePayload{}, err
	}
	host, err := d.string()
	if err != nil {
		return RoomStatePayload{}, err
	}
	count, err := d.u8()
	if err != nil {
		return RoomStatePayload{}, err
	}
	if count > maxDecodedPlayers {
		return RoomStatePayload{}, fmt.Errorf("player count %d exceeds limit", count)
	}
	players := make([]PlayerState, 0, int(count))
	for range count {
		id, err := d.string()
		if err != nil {
			return RoomStatePayload{}, err
		}
		name, err := d.string()
		if err != nil {
			return RoomStatePayload{}, err
		}
		character, err := d.string()
		if err != nil {
			return RoomStatePayload{}, err
		}
		flags, err := d.u8()
		if err != nil || flags&^uint8(3) != 0 {
			return RoomStatePayload{}, fmt.Errorf("invalid player flags %d", flags)
		}
		players = append(players, PlayerState{ID: id, DisplayName: name, CharacterID: character, Ready: flags&1 != 0, Connected: flags&2 != 0})
	}
	return RoomStatePayload{Status: status, HostPlayerID: host, Players: players}, nil
}

func (d *binaryDecoder) matchStarted() (MatchStartedPayload, error) {
	roomName, err := d.string()
	if err != nil {
		return MatchStartedPayload{}, err
	}
	mapID, err := d.string()
	if err != nil {
		return MatchStartedPayload{}, err
	}
	width, err := d.f32()
	if err != nil {
		return MatchStartedPayload{}, err
	}
	height, err := d.f32()
	if err != nil {
		return MatchStartedPayload{}, err
	}
	startedAt, err := d.i64()
	if err != nil {
		return MatchStartedPayload{}, err
	}
	count, err := d.u16()
	if err != nil {
		return MatchStartedPayload{}, err
	}
	if count > maxDecodedObstacles {
		return MatchStartedPayload{}, fmt.Errorf("obstacle count %d exceeds limit", count)
	}
	obstacles := make([]Obstacle, 0, int(count))
	for range count {
		id, err := d.string()
		if err != nil {
			return MatchStartedPayload{}, err
		}
		obstacleType, err := d.string()
		if err != nil {
			return MatchStartedPayload{}, err
		}
		x, err := d.f32()
		if err != nil {
			return MatchStartedPayload{}, err
		}
		y, err := d.f32()
		if err != nil {
			return MatchStartedPayload{}, err
		}
		radius, err := d.f32()
		if err != nil {
			return MatchStartedPayload{}, err
		}
		obstacles = append(obstacles, Obstacle{ID: id, Type: obstacleType, X: x, Y: y, Radius: radius})
	}
	return MatchStartedPayload{RoomName: roomName, MapID: mapID, MapWidth: width, MapHeight: height, StartedAtMs: startedAt, Obstacles: obstacles}, nil
}

func (d *binaryDecoder) snapshot() (SnapshotPayload, error) {
	tick, err := d.u32()
	if err != nil {
		return SnapshotPayload{}, err
	}
	serverTime, err := d.i64()
	if err != nil {
		return SnapshotPayload{}, err
	}
	playerCount, err := d.u8()
	if err != nil {
		return SnapshotPayload{}, err
	}
	if playerCount > maxDecodedPlayers {
		return SnapshotPayload{}, fmt.Errorf("player count %d exceeds limit", playerCount)
	}
	players := make([]SnapshotPlayer, 0, int(playerCount))
	for range playerCount {
		player, err := d.snapshotPlayer()
		if err != nil {
			return SnapshotPayload{}, err
		}
		players = append(players, player)
	}
	monsterCount, err := d.u16()
	if err != nil {
		return SnapshotPayload{}, err
	}
	if monsterCount > maxDecodedMonsters {
		return SnapshotPayload{}, fmt.Errorf("monster count %d exceeds limit", monsterCount)
	}
	monsters := make([]SnapshotMonster, 0, int(monsterCount))
	for range monsterCount {
		id, err := d.u32()
		if err != nil {
			return SnapshotPayload{}, err
		}
		x, err := d.f32()
		if err != nil {
			return SnapshotPayload{}, err
		}
		y, err := d.f32()
		if err != nil {
			return SnapshotPayload{}, err
		}
		hp, err := d.u16()
		if err != nil {
			return SnapshotPayload{}, err
		}
		maxHP, err := d.u16()
		if err != nil {
			return SnapshotPayload{}, err
		}
		monsters = append(monsters, SnapshotMonster{ID: uint64(id), X: x, Y: y, HP: int(hp), MaxHP: int(maxHP)})
	}
	pickupCount, err := d.u16()
	if err != nil {
		return SnapshotPayload{}, err
	}
	if pickupCount > maxDecodedPickups {
		return SnapshotPayload{}, fmt.Errorf("pickup count %d exceeds limit", pickupCount)
	}
	pickups := make([]SnapshotPickup, 0, int(pickupCount))
	for range pickupCount {
		id, err := d.u32()
		if err != nil {
			return SnapshotPayload{}, err
		}
		x, err := d.f32()
		if err != nil {
			return SnapshotPayload{}, err
		}
		y, err := d.f32()
		if err != nil {
			return SnapshotPayload{}, err
		}
		pickups = append(pickups, SnapshotPickup{ID: uint64(id), X: x, Y: y})
	}
	level, err := d.u16()
	if err != nil {
		return SnapshotPayload{}, err
	}
	experience, err := d.u16()
	if err != nil {
		return SnapshotPayload{}, err
	}
	required, err := d.u16()
	if err != nil {
		return SnapshotPayload{}, err
	}
	kills, err := d.u32()
	if err != nil {
		return SnapshotPayload{}, err
	}
	remaining, err := d.u32()
	if err != nil {
		return SnapshotPayload{}, err
	}
	return SnapshotPayload{
		Tick: uint64(tick), ServerTimeMs: serverTime, Players: players, Monsters: monsters, Pickups: pickups,
		Team:        SnapshotTeam{Level: int(level), Experience: int(experience), ExperienceRequired: int(required), TotalKills: int(kills)},
		RemainingMs: int64(remaining),
	}, nil
}

func (d *binaryDecoder) snapshotPlayer() (SnapshotPlayer, error) {
	id, err := d.string()
	if err != nil {
		return SnapshotPlayer{}, err
	}
	name, err := d.string()
	if err != nil {
		return SnapshotPlayer{}, err
	}
	values := make([]float64, 4)
	for index := range values {
		values[index], err = d.f32()
		if err != nil {
			return SnapshotPlayer{}, err
		}
	}
	flags, err := d.u8()
	if err != nil || flags&^uint8(3) != 0 {
		return SnapshotPlayer{}, fmt.Errorf("invalid snapshot player flags %d", flags)
	}
	hp, err := d.u16()
	if err != nil {
		return SnapshotPlayer{}, err
	}
	maxHP, err := d.u16()
	if err != nil {
		return SnapshotPlayer{}, err
	}
	lastInput, err := d.u32()
	if err != nil {
		return SnapshotPlayer{}, err
	}
	kills, err := d.u32()
	if err != nil {
		return SnapshotPlayer{}, err
	}
	facing := "right"
	if flags&1 != 0 {
		facing = "left"
	}
	return SnapshotPlayer{
		ID: id, DisplayName: name, X: values[0], Y: values[1], VelocityX: values[2], VelocityY: values[3],
		Facing: facing, HP: int(hp), MaxHP: int(maxHP), Alive: flags&2 != 0,
		LastProcessedInput: uint64(lastInput), Kills: int(kills),
	}, nil
}

func (d *binaryDecoder) projectileSpawned() (ProjectileSpawnedPayload, error) {
	id, err := d.u32()
	if err != nil {
		return ProjectileSpawnedPayload{}, err
	}
	owner, err := d.string()
	if err != nil {
		return ProjectileSpawnedPayload{}, err
	}
	weapon, err := d.string()
	if err != nil {
		return ProjectileSpawnedPayload{}, err
	}
	values := make([]float64, 4)
	for index := range values {
		values[index], err = d.f32()
		if err != nil {
			return ProjectileSpawnedPayload{}, err
		}
	}
	spawnTick, err := d.u32()
	return ProjectileSpawnedPayload{
		ProjectileID: uint64(id), OwnerID: owner, WeaponID: weapon,
		X: values[0], Y: values[1], VelocityX: values[2], VelocityY: values[3], SpawnTick: uint64(spawnTick),
	}, err
}

func encodeStatus(value string) (uint8, error) {
	switch value {
	case "lobby":
		return 0, nil
	case "running":
		return 1, nil
	case "finished":
		return 2, nil
	default:
		return 0, fmt.Errorf("unknown room status %q", value)
	}
}

func decodeStatus(value uint8) (string, error) {
	switch value {
	case 0:
		return "lobby", nil
	case 1:
		return "running", nil
	case 2:
		return "finished", nil
	default:
		return "", fmt.Errorf("unknown room status %d", value)
	}
}

func encodeRemovalReason(value string) (uint8, error) {
	switch value {
	case "enemy_hit":
		return 0, nil
	case "obstacle_hit":
		return 1, nil
	case "range_expired":
		return 2, nil
	case "match_ended":
		return 3, nil
	default:
		return 0, fmt.Errorf("unknown projectile removal reason %q", value)
	}
}

func decodeRemovalReason(value uint8) (string, error) {
	switch value {
	case 0:
		return "enemy_hit", nil
	case 1:
		return "obstacle_hit", nil
	case 2:
		return "range_expired", nil
	case 3:
		return "match_ended", nil
	default:
		return "", fmt.Errorf("unknown projectile removal reason %d", value)
	}
}

func encodeOutcome(value string) (uint8, error) {
	switch value {
	case "lost":
		return 0, nil
	case "won":
		return 1, nil
	default:
		return 0, fmt.Errorf("unknown match outcome %q", value)
	}
}

func decodeOutcome(value uint8) (string, error) {
	switch value {
	case 0:
		return "lost", nil
	case 1:
		return "won", nil
	default:
		return "", fmt.Errorf("unknown match outcome %d", value)
	}
}

func payloadTypeError(messageType MessageType, payload any) error {
	return fmt.Errorf("message type %d has incompatible payload %T", messageType, payload)
}

func uint32Value(value uint64, field string) (uint32, error) {
	if value > math.MaxUint32 {
		return 0, fmt.Errorf("%s exceeds uint32", field)
	}
	return uint32(value), nil
}

func nonNegativeUint32(value int64, field string) (uint32, error) {
	if value < 0 || value > math.MaxUint32 {
		return 0, fmt.Errorf("%s is outside uint32", field)
	}
	return uint32(value), nil
}

func nonNegativeUint32Int(value int, field string) (uint32, error) {
	if value < 0 || uint64(value) > math.MaxUint32 {
		return 0, fmt.Errorf("%s is outside uint32", field)
	}
	return uint32(value), nil
}

func nonNegativeUint16(value int, field string) (uint16, error) {
	if value < 0 || value > math.MaxUint16 {
		return 0, fmt.Errorf("%s is outside uint16", field)
	}
	return uint16(value), nil
}
