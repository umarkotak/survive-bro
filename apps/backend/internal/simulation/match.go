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
	ID                     string
	DisplayName            string
	CharacterID            string
	SpellID                string
	X                      float64
	Y                      float64
	VelocityX              float64
	VelocityY              float64
	MoveX                  float64
	MoveY                  float64
	Facing                 string
	HP                     int
	Alive                  bool
	LastInputAt            time.Time
	LastProcessedInput     uint64
	LastAttackAt           time.Duration
	Kills                  int
	MaxHP                  int
	ArmorPercent           float64
	MovementSpeed          float64
	HealthRegeneration     float64
	AttackBuffPercent      float64
	CooldownPercent        float64
	SpellDamage            int
	ProjectileSpeed        float64
	SpellBurst             int
	SpellDirections        int
	SpellCooldown          time.Duration
	SpellRange             float64
	ProjectileRadius       float64
	BaseMaxHP              int
	BaseArmorPercent       float64
	BaseMovementSpeed      float64
	BaseHealthRegeneration float64
	BaseAttackBuffPercent  float64
	BaseCooldownPercent    float64
	BaseSpellDamage        int
	BaseProjectileSpeed    float64
	BaseSpellBurst         int
	BaseSpellDirections    int
	regenAccumulator       float64
}

type Monster struct {
	ID            uint64
	TypeID        string
	X             float64
	Y             float64
	HP            int
	MaxHP         int
	Speed         float64
	Radius        float64
	ContactDamage int
	ContactDelay  time.Duration
	Experience    int
	Score         int
	LastContact   map[string]time.Duration
}

type Projectile struct {
	ID        uint64
	OwnerID   string
	X         float64
	Y         float64
	VelocityX float64
	VelocityY float64
	Travelled float64
	Damage    int
	Range     float64
	Radius    float64
}

type Pickup struct {
	ID    uint64
	Kind  string
	X     float64
	Y     float64
	Value int
}

type ProjectileRemoval struct {
	ID     uint64
	Reason string
}

type Events struct {
	SpawnedProjectiles []protocol.ProjectileSpawnedPayload
	RemovedProjectiles []ProjectileRemoval
	AppliedUpgrades    []protocol.UpgradeAppliedPayload
	MatchEnded         *protocol.MatchEndedPayload
	Metrics            TickMetrics
}

type TickMetrics struct {
	Total          time.Duration
	Movement       time.Duration
	Weapons        time.Duration
	ProjectileMove time.Duration
	BroadPhase     time.Duration
	NarrowPhase    time.Duration
	EnemyAI        time.Duration
	Pickups        time.Duration
	Spawning       time.Duration
	CandidatePairs uint64
	NarrowChecks   uint64
	ConfirmedHits  uint64
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
	EnemyScore       int
	Elapsed          time.Duration
	Finished         bool
	Level            LevelDefinition
	activeSpawn      SpawnRateDefinition
	nextLevelEvent   int
	spawnBudget      float64
	nextMonsterID    uint64
	nextProjectileID uint64
	nextPickupID     uint64
	rng              *rand.Rand
}

func NewMatch(startedAt time.Time, seed int64, levels ...LevelDefinition) *Match {
	level := levelOne
	if len(levels) > 0 {
		level = levels[0]
	}
	match := &Match{
		StartedAt:   startedAt,
		Players:     make(map[string]*Player),
		Monsters:    make(map[uint64]*Monster),
		Projectiles: make(map[uint64]*Projectile),
		Pickups:     make(map[uint64]*Pickup),
		TeamLevel:   1,
		Level:       level,
		rng:         rand.New(rand.NewSource(seed)),
	}
	match.processLevelEvents(&Events{})
	return match
}

func (m *Match) AddPlayer(id, displayName string, now time.Time, characterIDs ...string) *Player {
	characterID := "ranger"
	if len(characterIDs) > 0 && characterIDs[0] != "" {
		characterID = characterIDs[0]
	}
	character, ok := CharacterByID(characterID)
	if !ok {
		character = characters["ranger"]
	}
	spell, _ := SpellByID(character.BaseSpellID)
	order := len(m.Players)
	angle := float64(order) * (2 * math.Pi / 6)
	player := &Player{
		ID:                 id,
		DisplayName:        displayName,
		CharacterID:        character.ID,
		SpellID:            spell.ID,
		X:                  PlayerSpawnX + math.Cos(angle)*PlayerSpawnRadius,
		Y:                  PlayerSpawnY + math.Sin(angle)*PlayerSpawnRadius,
		Facing:             "right",
		HP:                 character.MaxHP,
		MaxHP:              character.MaxHP,
		ArmorPercent:       character.ArmorPercent,
		MovementSpeed:      character.MovementSpeed,
		HealthRegeneration: character.HealthRegeneration,
		AttackBuffPercent:  character.AttackBuffPercent,
		CooldownPercent:    character.CooldownPercent,
		SpellDamage:        spell.Damage,
		ProjectileSpeed:    spell.ProjectileSpeed,
		SpellBurst:         spell.Burst,
		SpellDirections:    spell.Directions,
		SpellCooldown:      spell.Cooldown,
		SpellRange:         spell.Range,
		ProjectileRadius:   spell.Radius,
		BaseMaxHP:          character.MaxHP, BaseArmorPercent: character.ArmorPercent, BaseMovementSpeed: character.MovementSpeed,
		BaseHealthRegeneration: character.HealthRegeneration, BaseAttackBuffPercent: character.AttackBuffPercent, BaseCooldownPercent: character.CooldownPercent,
		BaseSpellDamage: spell.Damage, BaseProjectileSpeed: spell.ProjectileSpeed, BaseSpellBurst: spell.Burst, BaseSpellDirections: spell.Directions,
		Alive:        true,
		LastInputAt:  now,
		LastAttackAt: -spell.Cooldown,
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

	tickStarted := time.Now()
	m.Tick++
	m.Elapsed += TickDuration
	events := Events{}
	phaseStarted := time.Now()
	m.updatePlayers(now)
	events.Metrics.Movement = time.Since(phaseStarted)
	phaseStarted = time.Now()
	m.updateWeapons(&events)
	events.Metrics.Weapons = time.Since(phaseStarted)
	m.updateProjectiles(&events, &events.Metrics)
	phaseStarted = time.Now()
	m.updateMonsters()
	events.Metrics.EnemyAI = time.Since(phaseStarted)
	phaseStarted = time.Now()
	m.updatePickups(&events)
	events.Metrics.Pickups = time.Since(phaseStarted)
	phaseStarted = time.Now()
	m.spawnMonsters()
	events.Metrics.Spawning = time.Since(phaseStarted)

	m.processLevelEvents(&events)
	if !m.Finished && !m.hasLivingPlayer() {
		m.finish("lost", &events)
	}
	events.Metrics.Total = time.Since(tickStarted)
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
			CharacterID:        player.CharacterID,
			X:                  player.X,
			Y:                  player.Y,
			VelocityX:          player.VelocityX,
			VelocityY:          player.VelocityY,
			MovementSpeed:      player.MovementSpeed,
			ArmorPercent:       player.ArmorPercent,
			HealthRegeneration: player.HealthRegeneration,
			AttackBuffPercent:  player.AttackBuffPercent,
			CooldownPercent:    player.CooldownPercent,
			SpellDamage:        player.SpellDamage,
			ProjectileSpeed:    player.ProjectileSpeed,
			SpellBurst:         player.SpellBurst,
			SpellDirections:    player.SpellDirections,
			Facing:             player.Facing,
			HP:                 player.HP,
			MaxHP:              player.MaxHP,
			Alive:              player.Alive,
			LastProcessedInput: player.LastProcessedInput,
			Kills:              player.Kills,
		})
	}

	monsters := make([]protocol.SnapshotMonster, 0, len(m.Monsters))
	monsterIDs := sortedUint64Keys(m.Monsters)
	for _, id := range monsterIDs {
		monster := m.Monsters[id]
		monsters = append(monsters, protocol.SnapshotMonster{ID: id, TypeID: monster.TypeID, X: monster.X, Y: monster.Y, HP: monster.HP, MaxHP: monster.MaxHP})
	}

	pickups := make([]protocol.SnapshotPickup, 0, len(m.Pickups))
	pickupIDs := sortedUint64Keys(m.Pickups)
	for _, id := range pickupIDs {
		pickup := m.Pickups[id]
		pickups = append(pickups, protocol.SnapshotPickup{ID: id, Kind: pickup.Kind, X: pickup.X, Y: pickup.Y})
	}

	remaining := max(time.Duration(0), m.Level.Duration-m.Elapsed)
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
		if player.HealthRegeneration > 0 && player.HP < player.MaxHP {
			player.regenAccumulator += player.HealthRegeneration * TickDuration.Seconds()
			heal := int(player.regenAccumulator)
			if heal > 0 {
				player.HP = min(player.MaxHP, player.HP+heal)
				player.regenAccumulator -= float64(heal)
			}
		}
		player.VelocityX = player.MoveX * player.MovementSpeed
		player.VelocityY = player.MoveY * player.MovementSpeed
		if player.MoveX < 0 {
			player.Facing = "left"
		} else if player.MoveX > 0 {
			player.Facing = "right"
		}
		player.X += player.VelocityX * TickDuration.Seconds()
		player.Y += player.VelocityY * TickDuration.Seconds()
		player.X, player.Y = resolveWorldAndObstacles(player.X, player.Y, PlayerRadius, m.Level.Obstacles)
	}
}

func (m *Match) updateWeapons(events *Events) {
	for _, player := range m.Players {
		cooldown := time.Duration(float64(player.SpellCooldown) * (1 - player.CooldownPercent))
		if !player.Alive || m.Elapsed-player.LastAttackAt < cooldown {
			continue
		}
		var target *Monster
		nearest := player.SpellRange
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
		baseAngle := math.Atan2(target.Y-player.Y, target.X-player.X)
		spread := ProjectileSpread * math.Pi / 180
		for direction := range player.SpellDirections {
			trajectory := baseAngle + (float64(direction)-float64(player.SpellDirections-1)/2)*spread
			for burst := range player.SpellBurst {
				angle := trajectory + (float64(burst)-float64(player.SpellBurst-1)/2)*(3*math.Pi/180)
				m.nextProjectileID++
				projectile := &Projectile{
					ID:        m.nextProjectileID,
					OwnerID:   player.ID,
					X:         player.X,
					Y:         player.Y,
					VelocityX: math.Cos(angle) * player.ProjectileSpeed,
					VelocityY: math.Sin(angle) * player.ProjectileSpeed,
					Damage:    int(math.Round(float64(player.SpellDamage) * (1 + player.AttackBuffPercent))),
					Range:     player.SpellRange,
					Radius:    player.ProjectileRadius,
				}
				m.Projectiles[projectile.ID] = projectile
				events.SpawnedProjectiles = append(events.SpawnedProjectiles, protocol.ProjectileSpawnedPayload{
					ProjectileID: projectile.ID,
					OwnerID:      player.ID,
					WeaponID:     player.SpellID,
					X:            projectile.X,
					Y:            projectile.Y,
					VelocityX:    projectile.VelocityX,
					VelocityY:    projectile.VelocityY,
					SpawnTick:    m.Tick,
				})
			}
		}
		player.LastAttackAt = m.Elapsed
	}
}

func (m *Match) updateProjectiles(events *Events, metrics *TickMetrics) {
	moveStarted := time.Now()
	for id, projectile := range m.Projectiles {
		stepX := projectile.VelocityX * TickDuration.Seconds()
		stepY := projectile.VelocityY * TickDuration.Seconds()
		projectile.X += stepX
		projectile.Y += stepY
		projectile.Travelled += math.Hypot(stepX, stepY)
		if projectile.Travelled >= projectile.Range {
			m.removeProjectile(id, "range_expired", events)
		}
	}
	metrics.ProjectileMove = time.Since(moveStarted)

	narrowStarted := time.Now()
	for id, projectile := range m.Projectiles {
		obstacleHit, obstacleChecks := collidesObstacleCounted(projectile.X, projectile.Y, projectile.Radius, m.Level.Obstacles)
		metrics.CandidatePairs += obstacleChecks
		metrics.NarrowChecks += obstacleChecks
		if obstacleHit {
			metrics.ConfirmedHits++
			m.removeProjectile(id, "obstacle_hit", events)
			continue
		}
		for monsterID, monster := range m.Monsters {
			metrics.CandidatePairs++
			metrics.NarrowChecks++
			if !overlaps(projectile.X, projectile.Y, projectile.Radius, monster.X, monster.Y, monster.Radius) {
				continue
			}
			monster.HP -= projectile.Damage
			metrics.ConfirmedHits++
			m.removeProjectile(id, "enemy_hit", events)
			if monster.HP <= 0 {
				m.killMonster(monsterID, projectile.OwnerID)
			}
			break
		}
	}
	metrics.NarrowPhase = time.Since(narrowStarted)
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
			attemptX := monster.X + dx/distance*monster.Speed*TickDuration.Seconds()
			attemptY := monster.Y + dy/distance*monster.Speed*TickDuration.Seconds()
			resolvedX, resolvedY := resolveWorldAndObstacles(attemptX, attemptY, monster.Radius, m.Level.Obstacles)
			if resolvedX != attemptX || resolvedY != attemptY {
				resolvedX += -dy / distance * monster.Speed * TickDuration.Seconds() * 0.55
				resolvedY += dx / distance * monster.Speed * TickDuration.Seconds() * 0.55
				resolvedX, resolvedY = resolveWorldAndObstacles(resolvedX, resolvedY, monster.Radius, m.Level.Obstacles)
			}
			monster.X = resolvedX
			monster.Y = resolvedY
		}

		for _, player := range m.Players {
			if !player.Alive || !overlaps(monster.X, monster.Y, monster.Radius, player.X, player.Y, PlayerRadius) {
				continue
			}
			last := monster.LastContact[player.ID]
			if m.Elapsed-last < monster.ContactDelay {
				continue
			}
			monster.LastContact[player.ID] = m.Elapsed
			damage := max(1, int(math.Round(float64(monster.ContactDamage)*(1-player.ArmorPercent))))
			player.HP = max(0, player.HP-damage)
			if player.HP == 0 {
				player.Alive = false
				player.MoveX = 0
				player.MoveY = 0
			}
		}
	}
}

func (m *Match) updatePickups(events *Events) {
	for pickupID, pickup := range m.Pickups {
		if pickup.Kind == "power_crate" {
			collector := m.nearestLivingPlayerWithin(pickup.X, pickup.Y, PowerCrateRadius)
			if collector == nil {
				continue
			}
			delete(m.Pickups, pickupID)
			events.AppliedUpgrades = append(events.AppliedUpgrades, m.applyRandomUpgrade(collector, "treasure_chest"))
			continue
		}

		player := m.nearestLivingPlayerWithin(pickup.X, pickup.Y, PlayerPickupRadius)
		if player == nil {
			continue
		}
		dx := player.X - pickup.X
		dy := player.Y - pickup.Y
		distance := math.Hypot(dx, dy)
		if distance > 0 {
			step := min(distance, PickupAttractSpeed*TickDuration.Seconds())
			pickup.X += dx / distance * step
			pickup.Y += dy / distance * step
			distance -= step
		}
		if distance > PickupCollectRadius {
			continue
		}
		delete(m.Pickups, pickupID)
		m.TeamExperience += max(1, pickup.Value)
		for m.TeamExperience >= RequiredExperience(m.TeamLevel) {
			m.TeamExperience -= RequiredExperience(m.TeamLevel)
			m.TeamLevel++
			for _, id := range sortedPlayerIDs(m.Players) {
				events.AppliedUpgrades = append(events.AppliedUpgrades, m.applyRandomUpgrade(m.Players[id], "level_up"))
			}
		}
	}
}

func (m *Match) spawnMonsters() {
	multiplier := 1 + 0.55*float64(len(m.Players)-1)
	maximum := int(math.Round(float64(m.activeSpawn.MaxLiving) * multiplier))
	if maximum == 0 || len(m.activeSpawn.Entries) == 0 {
		return
	}
	if len(m.Monsters) >= maximum {
		return
	}
	m.spawnBudget += m.activeSpawn.RatePerSecond * multiplier * TickDuration.Seconds()
	for m.spawnBudget >= 1 && len(m.Monsters) < maximum {
		m.spawnBudget--
		m.spawnMonsterOf(m.selectSpawnEnemy())
	}
}

func (m *Match) spawnMonster() {
	m.spawnMonsterOf(m.selectSpawnEnemy())
}

func (m *Match) selectSpawnEnemy() string {
	total := 0
	for _, entry := range m.activeSpawn.Entries {
		total += max(0, entry.Weight)
	}
	if total == 0 {
		return ""
	}
	roll := m.rng.Intn(total)
	for _, entry := range m.activeSpawn.Entries {
		roll -= max(0, entry.Weight)
		if roll < 0 {
			return entry.EnemyID
		}
	}
	return ""
}

func (m *Match) spawnMonsterOf(enemyID string) {
	definition, ok := EnemyByID(enemyID)
	if !ok {
		return
	}
	living := m.livingPlayers()
	if len(living) == 0 {
		return
	}
	target := living[m.rng.Intn(len(living))]
	for range 10 {
		angle := m.rng.Float64() * 2 * math.Pi
		distance := 700 + m.rng.Float64()*200
		x := clamp(target.X+math.Cos(angle)*distance, definition.Radius, WorldWidth-definition.Radius)
		y := clamp(target.Y+math.Sin(angle)*distance, definition.Radius, WorldHeight-definition.Radius)
		if collidesObstacle(x, y, definition.Radius, m.Level.Obstacles) {
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
		m.Monsters[m.nextMonsterID] = &Monster{ID: m.nextMonsterID, TypeID: definition.ID, X: x, Y: y, HP: definition.MaxHP, MaxHP: definition.MaxHP, Speed: definition.Speed, Radius: definition.Radius, ContactDamage: definition.ContactDamage, ContactDelay: definition.ContactDelay, Experience: definition.Experience, Score: definition.Score, LastContact: make(map[string]time.Duration)}
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
	m.Pickups[m.nextPickupID] = &Pickup{ID: m.nextPickupID, Kind: "experience", X: monster.X, Y: monster.Y, Value: monster.Experience}
	m.TotalKills++
	m.EnemyScore += monster.Score
	if m.TotalKills%PowerCrateEveryKills == 0 {
		m.nextPickupID++
		x, y := resolveWorldAndObstacles(monster.X+44, monster.Y, 18, m.Level.Obstacles)
		m.Pickups[m.nextPickupID] = &Pickup{ID: m.nextPickupID, Kind: "power_crate", X: x, Y: y}
	}
	if owner, ok := m.Players[ownerID]; ok {
		owner.Kills++
	}
}

func (m *Match) applyRandomUpgrade(player *Player, source string) protocol.UpgradeAppliedPayload {
	available := []string{"max_health", "health_regeneration", "attack_buff", "spell_damage", "projectile_speed"}
	if player.ArmorPercent < MaximumArmorPercent {
		available = append(available, "armor")
	}
	if player.MovementSpeed < player.BaseMovementSpeed*(1+MaximumMovementBonus) {
		available = append(available, "movement_speed")
	}
	if player.CooldownPercent < 0.60 {
		available = append(available, "cooldown")
	}
	if player.SpellBurst < 2 {
		available = append(available, "spell_burst")
	}
	if player.SpellDirections < 4 {
		available = append(available, "spell_directions")
	}
	attribute := available[m.rng.Intn(len(available))]
	baseValue, addedValue, finalValue := 0.0, 0.0, 0.0
	switch attribute {
	case "max_health":
		baseValue, addedValue = float64(player.BaseMaxHP), 20
		player.MaxHP += 20
		player.HP += 20
		finalValue = float64(player.MaxHP)
	case "armor":
		baseValue, addedValue = player.BaseArmorPercent, 0.05
		player.ArmorPercent = min(MaximumArmorPercent, player.ArmorPercent+0.05)
		finalValue = player.ArmorPercent
	case "movement_speed":
		baseValue, addedValue = player.BaseMovementSpeed, player.BaseMovementSpeed*0.08
		player.MovementSpeed = min(player.BaseMovementSpeed*(1+MaximumMovementBonus), player.MovementSpeed+player.BaseMovementSpeed*0.08)
		finalValue = player.MovementSpeed
	case "health_regeneration":
		baseValue, addedValue = player.BaseHealthRegeneration, 1
		player.HealthRegeneration += 1
		finalValue = player.HealthRegeneration
	case "attack_buff":
		baseValue, addedValue = player.BaseAttackBuffPercent, 0.10
		player.AttackBuffPercent += 0.10
		finalValue = player.AttackBuffPercent
	case "cooldown":
		baseValue, addedValue = player.BaseCooldownPercent, 0.08
		player.CooldownPercent = min(0.60, player.CooldownPercent+0.08)
		finalValue = player.CooldownPercent
	case "spell_damage":
		baseValue, addedValue = float64(player.BaseSpellDamage), 4
		player.SpellDamage += 4
		finalValue = float64(player.SpellDamage)
	case "projectile_speed":
		baseValue, addedValue = player.BaseProjectileSpeed, 70
		player.ProjectileSpeed += 70
		finalValue = player.ProjectileSpeed
	case "spell_burst":
		baseValue, addedValue = float64(player.BaseSpellBurst), 1
		player.SpellBurst++
		finalValue = float64(player.SpellBurst)
	case "spell_directions":
		baseValue, addedValue = float64(player.BaseSpellDirections), 1
		player.SpellDirections++
		finalValue = float64(player.SpellDirections)
	}
	return protocol.UpgradeAppliedPayload{PlayerID: player.ID, Source: source, Attribute: attribute, BaseValue: baseValue, AddedValue: addedValue, FinalValue: finalValue}
}

func sortedPlayerIDs(players map[string]*Player) []string {
	ids := make([]string, 0, len(players))
	for id := range players {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (m *Match) nearestLivingPlayerWithin(x, y, maximumDistance float64) *Player {
	var nearest *Player
	nearestDistance := maximumDistance
	for _, player := range m.Players {
		if !player.Alive {
			continue
		}
		distance := math.Hypot(player.X-x, player.Y-y)
		if distance <= nearestDistance {
			nearest = player
			nearestDistance = distance
		}
	}
	return nearest
}

func (m *Match) processLevelEvents(events *Events) {
	for m.nextLevelEvent < len(m.Level.Events) && m.Elapsed >= m.Level.Events[m.nextLevelEvent].At {
		event := m.Level.Events[m.nextLevelEvent]
		m.nextLevelEvent++
		switch event.Type {
		case "spawn_rate":
			if event.SpawnRate != nil {
				m.activeSpawn = *event.SpawnRate
			}
		case "boss":
			m.spawnMonsterOf(event.EnemyID)
		case "end":
			m.finish("won", events)
		}
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
		Score:      m.EnemyScore + m.TeamLevel*250 + int(m.Elapsed/time.Second),
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

func resolveWorldAndObstacles(x, y, radius float64, obstacles ...[]Obstacle) (float64, float64) {
	activeObstacles := Obstacles
	if len(obstacles) > 0 {
		activeObstacles = obstacles[0]
	}
	x = clamp(x, radius, WorldWidth-radius)
	y = clamp(y, radius, WorldHeight-radius)
	for _, obstacle := range activeObstacles {
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

func collidesObstacle(x, y, radius float64, obstacles ...[]Obstacle) bool {
	collides, _ := collidesObstacleCounted(x, y, radius, obstacles...)
	return collides
}

func collidesObstacleCounted(x, y, radius float64, obstacles ...[]Obstacle) (bool, uint64) {
	activeObstacles := Obstacles
	if len(obstacles) > 0 {
		activeObstacles = obstacles[0]
	}
	var checks uint64
	for _, obstacle := range activeObstacles {
		checks++
		if overlaps(x, y, radius, obstacle.X, obstacle.Y, obstacle.Radius) {
			return true, checks
		}
	}
	return false, checks
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
