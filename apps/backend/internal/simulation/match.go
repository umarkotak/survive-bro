package simulation

import (
	"errors"
	"math"
	"math/rand"
	"sort"
	"time"

	"survive-bro/apps/backend/internal/protocol"
)

var ErrInvalidInput = errors.New("movement input is invalid")

type Player struct {
	ID                 string
	DisplayName        string
	X                  float64
	Y                  float64
	VelocityX          float64
	VelocityY          float64
	MoveX              float64
	MoveY              float64
	Facing             string
	HP                 int
	Alive              bool
	LastInputAt        time.Time
	LastProcessedInput uint64
	LastAttackAt       time.Duration
	Kills              int
}

type Monster struct {
	ID          uint64
	X           float64
	Y           float64
	HP          int
	LastContact map[string]time.Duration
}

type Projectile struct {
	ID        uint64
	OwnerID   string
	X         float64
	Y         float64
	VelocityX float64
	VelocityY float64
	Travelled float64
}

type Pickup struct {
	ID uint64
	X  float64
	Y  float64
}

type ProjectileRemoval struct {
	ID     uint64
	Reason string
}

type Events struct {
	SpawnedProjectiles []protocol.ProjectileSpawnedPayload
	RemovedProjectiles []ProjectileRemoval
	MatchEnded         *protocol.MatchEndedPayload
}

type Match struct {
	StartedAt        time.Time
	Tick             uint64
	Players          map[string]*Player
	Monsters         map[uint64]*Monster
	Projectiles      map[uint64]*Projectile
	Pickups          map[uint64]*Pickup
	TeamLevel        int
	TeamExperience   int
	TotalKills       int
	Elapsed          time.Duration
	Finished         bool
	spawnBudget      float64
	nextMonsterID    uint64
	nextProjectileID uint64
	nextPickupID     uint64
	rng              *rand.Rand
}

func NewMatch(startedAt time.Time, seed int64) *Match {
	return &Match{
		StartedAt:   startedAt,
		Players:     make(map[string]*Player),
		Monsters:    make(map[uint64]*Monster),
		Projectiles: make(map[uint64]*Projectile),
		Pickups:     make(map[uint64]*Pickup),
		TeamLevel:   1,
		rng:         rand.New(rand.NewSource(seed)),
	}
}

func (m *Match) AddPlayer(id, displayName string, now time.Time) *Player {
	order := len(m.Players)
	angle := float64(order) * (2 * math.Pi / 4)
	player := &Player{
		ID:           id,
		DisplayName:  displayName,
		X:            PlayerSpawnX + math.Cos(angle)*PlayerSpawnRadius,
		Y:            PlayerSpawnY + math.Sin(angle)*PlayerSpawnRadius,
		Facing:       "right",
		HP:           PlayerMaxHP,
		Alive:        true,
		LastInputAt:  now,
		LastAttackAt: -WeaponCooldown,
	}
	m.Players[id] = player
	return player
}

func (m *Match) RemovePlayer(id string) {
	delete(m.Players, id)
}

func (m *Match) ApplyInput(playerID string, input protocol.InputPayload, now time.Time) error {
	player, ok := m.Players[playerID]
	if !ok || !player.Alive || m.Finished {
		return nil
	}
	if math.IsNaN(input.MoveX) || math.IsNaN(input.MoveY) || math.IsInf(input.MoveX, 0) || math.IsInf(input.MoveY, 0) || input.MoveX < -1 || input.MoveX > 1 || input.MoveY < -1 || input.MoveY > 1 {
		return ErrInvalidInput
	}
	if input.Sequence <= player.LastProcessedInput {
		return nil
	}
	length := math.Hypot(input.MoveX, input.MoveY)
	if length > 1 {
		input.MoveX /= length
		input.MoveY /= length
	}
	player.MoveX = input.MoveX
	player.MoveY = input.MoveY
	player.LastInputAt = now
	player.LastProcessedInput = input.Sequence
	return nil
}

func (m *Match) Step(now time.Time) Events {
	if m.Finished || len(m.Players) == 0 {
		return Events{}
	}

	m.Tick++
	m.Elapsed += TickDuration
	events := Events{}
	m.updatePlayers(now)
	m.updateWeapons(&events)
	m.updateProjectiles(&events)
	m.updateMonsters()
	m.collectPickups()
	m.spawnMonsters()

	if m.Elapsed >= MatchDuration {
		m.finish("won", &events)
	} else if !m.hasLivingPlayer() {
		m.finish("lost", &events)
	}
	return events
}

func (m *Match) Snapshot(now time.Time) protocol.SnapshotPayload {
	players := make([]protocol.SnapshotPlayer, 0, len(m.Players))
	playerIDs := make([]string, 0, len(m.Players))
	for id := range m.Players {
		playerIDs = append(playerIDs, id)
	}
	sort.Strings(playerIDs)
	for _, id := range playerIDs {
		player := m.Players[id]
		players = append(players, protocol.SnapshotPlayer{
			ID:                 player.ID,
			DisplayName:        player.DisplayName,
			X:                  player.X,
			Y:                  player.Y,
			VelocityX:          player.VelocityX,
			VelocityY:          player.VelocityY,
			Facing:             player.Facing,
			HP:                 player.HP,
			MaxHP:              PlayerMaxHP,
			Alive:              player.Alive,
			LastProcessedInput: player.LastProcessedInput,
			Kills:              player.Kills,
		})
	}

	monsters := make([]protocol.SnapshotMonster, 0, len(m.Monsters))
	monsterIDs := sortedUint64Keys(m.Monsters)
	for _, id := range monsterIDs {
		monster := m.Monsters[id]
		monsters = append(monsters, protocol.SnapshotMonster{ID: id, X: monster.X, Y: monster.Y, HP: monster.HP, MaxHP: MonsterMaxHP})
	}

	pickups := make([]protocol.SnapshotPickup, 0, len(m.Pickups))
	pickupIDs := sortedUint64Keys(m.Pickups)
	for _, id := range pickupIDs {
		pickup := m.Pickups[id]
		pickups = append(pickups, protocol.SnapshotPickup{ID: id, X: pickup.X, Y: pickup.Y})
	}

	remaining := max(time.Duration(0), MatchDuration-m.Elapsed)
	return protocol.SnapshotPayload{
		Tick:         m.Tick,
		ServerTimeMs: now.UnixMilli(),
		Players:      players,
		Monsters:     monsters,
		Pickups:      pickups,
		Team: protocol.SnapshotTeam{
			Level:              m.TeamLevel,
			Experience:         m.TeamExperience,
			ExperienceRequired: RequiredExperience(m.TeamLevel),
			TotalKills:         m.TotalKills,
		},
		RemainingMs: remaining.Milliseconds(),
	}
}

func (m *Match) updatePlayers(now time.Time) {
	for _, player := range m.Players {
		if !player.Alive {
			player.VelocityX = 0
			player.VelocityY = 0
			continue
		}
		if now.Sub(player.LastInputAt) > InputStaleAfter {
			player.MoveX = 0
			player.MoveY = 0
		}
		player.VelocityX = player.MoveX * PlayerSpeed
		player.VelocityY = player.MoveY * PlayerSpeed
		if player.MoveX < 0 {
			player.Facing = "left"
		} else if player.MoveX > 0 {
			player.Facing = "right"
		}
		player.X += player.VelocityX * TickDuration.Seconds()
		player.Y += player.VelocityY * TickDuration.Seconds()
		player.X, player.Y = resolveWorldAndObstacles(player.X, player.Y, PlayerRadius)
	}
}

func (m *Match) updateWeapons(events *Events) {
	for _, player := range m.Players {
		if !player.Alive || m.Elapsed-player.LastAttackAt < WeaponCooldown {
			continue
		}
		var target *Monster
		nearest := ProjectileRange
		for _, monster := range m.Monsters {
			distance := math.Hypot(monster.X-player.X, monster.Y-player.Y)
			if distance <= nearest {
				nearest = distance
				target = monster
			}
		}
		if target == nil {
			continue
		}
		m.nextProjectileID++
		angle := math.Atan2(target.Y-player.Y, target.X-player.X)
		projectileSpeed := ProjectileSpeedAtLevel(m.TeamLevel)
		projectile := &Projectile{
			ID:        m.nextProjectileID,
			OwnerID:   player.ID,
			X:         player.X,
			Y:         player.Y,
			VelocityX: math.Cos(angle) * projectileSpeed,
			VelocityY: math.Sin(angle) * projectileSpeed,
		}
		m.Projectiles[projectile.ID] = projectile
		player.LastAttackAt = m.Elapsed
		events.SpawnedProjectiles = append(events.SpawnedProjectiles, protocol.ProjectileSpawnedPayload{
			ProjectileID: projectile.ID,
			OwnerID:      player.ID,
			WeaponID:     "arc_bolt",
			X:            projectile.X,
			Y:            projectile.Y,
			VelocityX:    projectile.VelocityX,
			VelocityY:    projectile.VelocityY,
			SpawnTick:    m.Tick,
		})
	}
}

func (m *Match) updateProjectiles(events *Events) {
	for id, projectile := range m.Projectiles {
		stepX := projectile.VelocityX * TickDuration.Seconds()
		stepY := projectile.VelocityY * TickDuration.Seconds()
		projectile.X += stepX
		projectile.Y += stepY
		projectile.Travelled += math.Hypot(stepX, stepY)
		if projectile.Travelled >= ProjectileRange {
			m.removeProjectile(id, "range_expired", events)
			continue
		}
		if collidesObstacle(projectile.X, projectile.Y, ProjectileRadius) {
			m.removeProjectile(id, "obstacle_hit", events)
			continue
		}
		for monsterID, monster := range m.Monsters {
			if !overlaps(projectile.X, projectile.Y, ProjectileRadius, monster.X, monster.Y, MonsterRadius) {
				continue
			}
			monster.HP -= WeaponDamage
			m.removeProjectile(id, "enemy_hit", events)
			if monster.HP <= 0 {
				m.killMonster(monsterID, projectile.OwnerID)
			}
			break
		}
	}
}

func (m *Match) updateMonsters() {
	for _, monster := range m.Monsters {
		target := m.nearestLivingPlayer(monster.X, monster.Y)
		if target == nil {
			return
		}
		dx := target.X - monster.X
		dy := target.Y - monster.Y
		distance := math.Hypot(dx, dy)
		if distance > 0 {
			attemptX := monster.X + dx/distance*MonsterSpeed*TickDuration.Seconds()
			attemptY := monster.Y + dy/distance*MonsterSpeed*TickDuration.Seconds()
			resolvedX, resolvedY := resolveWorldAndObstacles(attemptX, attemptY, MonsterRadius)
			if resolvedX != attemptX || resolvedY != attemptY {
				resolvedX += -dy / distance * MonsterSpeed * TickDuration.Seconds() * 0.55
				resolvedY += dx / distance * MonsterSpeed * TickDuration.Seconds() * 0.55
				resolvedX, resolvedY = resolveWorldAndObstacles(resolvedX, resolvedY, MonsterRadius)
			}
			monster.X = resolvedX
			monster.Y = resolvedY
		}

		for _, player := range m.Players {
			if !player.Alive || !overlaps(monster.X, monster.Y, MonsterRadius, player.X, player.Y, PlayerRadius) {
				continue
			}
			last := monster.LastContact[player.ID]
			if m.Elapsed-last < MonsterContactDelay {
				continue
			}
			monster.LastContact[player.ID] = m.Elapsed
			player.HP = max(0, player.HP-MonsterContactDamage)
			if player.HP == 0 {
				player.Alive = false
				player.MoveX = 0
				player.MoveY = 0
			}
		}
	}
}

func (m *Match) collectPickups() {
	for pickupID, pickup := range m.Pickups {
		collected := false
		for _, player := range m.Players {
			if player.Alive && math.Hypot(pickup.X-player.X, pickup.Y-player.Y) <= PlayerPickupRadius {
				collected = true
				break
			}
		}
		if !collected {
			continue
		}
		delete(m.Pickups, pickupID)
		m.TeamExperience += MonsterXP
		for m.TeamExperience >= RequiredExperience(m.TeamLevel) {
			m.TeamExperience -= RequiredExperience(m.TeamLevel)
			m.TeamLevel++
		}
	}
}

func (m *Match) spawnMonsters() {
	baseRate, baseMax := difficulty(m.Elapsed)
	multiplier := 1 + 0.55*float64(len(m.Players)-1)
	maximum := int(math.Round(float64(baseMax) * multiplier))
	if len(m.Monsters) >= maximum {
		return
	}
	m.spawnBudget += baseRate * multiplier * TickDuration.Seconds()
	for m.spawnBudget >= 1 && len(m.Monsters) < maximum {
		m.spawnBudget--
		m.spawnMonster()
	}
}

func (m *Match) spawnMonster() {
	living := m.livingPlayers()
	if len(living) == 0 {
		return
	}
	target := living[m.rng.Intn(len(living))]
	for range 10 {
		angle := m.rng.Float64() * 2 * math.Pi
		distance := 700 + m.rng.Float64()*200
		x := clamp(target.X+math.Cos(angle)*distance, MonsterRadius, WorldWidth-MonsterRadius)
		y := clamp(target.Y+math.Sin(angle)*distance, MonsterRadius, WorldHeight-MonsterRadius)
		if collidesObstacle(x, y, MonsterRadius) {
			continue
		}
		valid := true
		for _, player := range living {
			if math.Hypot(x-player.X, y-player.Y) < 600 {
				valid = false
				break
			}
		}
		if !valid {
			continue
		}
		m.nextMonsterID++
		m.Monsters[m.nextMonsterID] = &Monster{ID: m.nextMonsterID, X: x, Y: y, HP: MonsterMaxHP, LastContact: make(map[string]time.Duration)}
		return
	}
}

func (m *Match) removeProjectile(id uint64, reason string, events *Events) {
	delete(m.Projectiles, id)
	events.RemovedProjectiles = append(events.RemovedProjectiles, ProjectileRemoval{ID: id, Reason: reason})
}

func (m *Match) killMonster(monsterID uint64, ownerID string) {
	monster := m.Monsters[monsterID]
	delete(m.Monsters, monsterID)
	m.nextPickupID++
	m.Pickups[m.nextPickupID] = &Pickup{ID: m.nextPickupID, X: monster.X, Y: monster.Y}
	m.TotalKills++
	if owner, ok := m.Players[ownerID]; ok {
		owner.Kills++
	}
}

func (m *Match) finish(outcome string, events *Events) {
	m.Finished = true
	for id := range m.Projectiles {
		m.removeProjectile(id, "match_ended", events)
	}
	events.MatchEnded = &protocol.MatchEndedPayload{
		Outcome:    outcome,
		SurvivalMs: m.Elapsed.Milliseconds(),
		TeamLevel:  m.TeamLevel,
		TotalKills: m.TotalKills,
	}
}

func (m *Match) hasLivingPlayer() bool {
	for _, player := range m.Players {
		if player.Alive {
			return true
		}
	}
	return false
}

func (m *Match) livingPlayers() []*Player {
	players := make([]*Player, 0, len(m.Players))
	for _, player := range m.Players {
		if player.Alive {
			players = append(players, player)
		}
	}
	return players
}

func (m *Match) nearestLivingPlayer(x, y float64) *Player {
	var nearest *Player
	nearestDistance := math.MaxFloat64
	for _, player := range m.Players {
		if !player.Alive {
			continue
		}
		distance := math.Hypot(player.X-x, player.Y-y)
		if distance < nearestDistance {
			nearestDistance = distance
			nearest = player
		}
	}
	return nearest
}

func RequiredExperience(level int) int {
	return int(math.Round(8 + 5*math.Pow(float64(level), 1.45)))
}

func difficulty(elapsed time.Duration) (float64, int) {
	switch {
	case elapsed < time.Minute:
		return 1, 60
	case elapsed < 150*time.Second:
		return 1.8, 110
	case elapsed < 240*time.Second:
		return 2.7, 170
	default:
		return 3.5, 240
	}
}

func resolveWorldAndObstacles(x, y, radius float64) (float64, float64) {
	x = clamp(x, radius, WorldWidth-radius)
	y = clamp(y, radius, WorldHeight-radius)
	for _, obstacle := range Obstacles {
		dx := x - obstacle.X
		dy := y - obstacle.Y
		minimum := radius + obstacle.Radius
		distance := math.Hypot(dx, dy)
		if distance >= minimum {
			continue
		}
		if distance == 0 {
			x = obstacle.X + minimum
			continue
		}
		scale := minimum / distance
		x = obstacle.X + dx*scale
		y = obstacle.Y + dy*scale
	}
	return clamp(x, radius, WorldWidth-radius), clamp(y, radius, WorldHeight-radius)
}

func collidesObstacle(x, y, radius float64) bool {
	for _, obstacle := range Obstacles {
		if overlaps(x, y, radius, obstacle.X, obstacle.Y, obstacle.Radius) {
			return true
		}
	}
	return false
}

func overlaps(ax, ay, ar, bx, by, br float64) bool {
	dx := ax - bx
	dy := ay - by
	radii := ar + br
	return dx*dx+dy*dy < radii*radii
}

func clamp(value, minimum, maximum float64) float64 {
	return math.Max(minimum, math.Min(maximum, value))
}

func sortedUint64Keys[T any](values map[uint64]*T) []uint64 {
	keys := make([]uint64, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}
