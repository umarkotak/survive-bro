package simulation

import (
	"errors"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"time"

	"survive-bro/apps/backend/internal/protocol"
)

var (
	ErrInvalidInput      = errors.New("movement input is invalid")
	ErrNoUpgradePhase    = errors.New("no upgrade phase is active")
	ErrStaleUpgradeOffer = errors.New("upgrade offer is stale")
	ErrInvalidUpgrade    = errors.New("upgrade choice is invalid")
	ErrUpgradeSelected   = errors.New("upgrade choice was already selected")
	ErrDebugLevelUp      = errors.New("debug level up is unavailable")
)

const UpgradeSelectionTimeout = 50 * time.Second

type Player struct {
	ID                           string
	DisplayName                  string
	CharacterID                  string
	SpellIDs                     []string
	SpellLevels                  map[string]int
	SpellID                      string
	X                            float64
	Y                            float64
	VelocityX                    float64
	VelocityY                    float64
	MoveX                        float64
	MoveY                        float64
	Facing                       string
	HP                           int
	Alive                        bool
	LastInputAt                  time.Time
	LastProcessedInput           uint64
	LastAttackAt                 time.Duration
	LastSpellAttackAt            map[string]time.Duration
	Kills                        int
	MaxHP                        int
	ArmorPercent                 float64
	MovementSpeed                float64
	HealthRegeneration           float64
	AttackBuffPercent            float64
	CooldownPercent              float64
	SpellDamage                  int
	ProjectileSpeed              float64
	SpellBurst                   int
	SpellDirections              int
	SpellKind                    string
	BeamLength                   float64
	BeamWidth                    float64
	SpellDuration                time.Duration
	DamageInterval               time.Duration
	ExplosionRadius              float64
	ExplosionDuration            time.Duration
	ImpactDamage                 int
	SpellCooldown                time.Duration
	SpellRange                   float64
	ProjectileRadius             float64
	BaseMaxHP                    int
	BaseArmorPercent             float64
	BaseMovementSpeed            float64
	BaseHealthRegeneration       float64
	BaseAttackBuffPercent        float64
	BaseCooldownPercent          float64
	BaseSpellDamage              int
	BaseProjectileSpeed          float64
	BaseSpellBurst               int
	BaseSpellDirections          int
	ResurrectionDuration         time.Duration
	ResurrectionRadius           float64
	ResurrectionImmunityDuration time.Duration
	ResurrectionProgress         time.Duration
	ResurrectionPending          bool
	ImmuneUntil                  time.Duration
	regenAccumulator             float64
}

type Monster struct {
	ID                     uint64
	TypeID                 string
	X                      float64
	Y                      float64
	HP                     int
	MaxHP                  int
	Speed                  float64
	Radius                 float64
	ContactDamage          int
	Armor                  int
	ContactDelay           time.Duration
	Experience             int
	Score                  int
	LastContact            map[string]time.Duration
	SpellIDs               []string
	LastSpellAt            map[string]time.Duration
	AttackDamageMultiplier float64
	MovementMode           string
	DashInterval           time.Duration
	DashWindup             time.Duration
	DashDuration           time.Duration
	DashSpeedMultiplier    float64
	DashState              string
	DashTimer              time.Duration
	DashDirX               float64
	DashDirY               float64
}

type Projectile struct {
	ID                uint64
	OwnerID           string
	X                 float64
	Y                 float64
	VelocityX         float64
	VelocityY         float64
	Travelled         float64
	Damage            int
	Range             float64
	Radius            float64
	SpellID           string
	ExplosionRadius   float64
	ExplosionDuration time.Duration
	DamageInterval    time.Duration
	ImpactDamage      int
	TargetsPlayers    bool
}

type Beam struct {
	ID             uint64
	OwnerID        string
	SpellID        string
	X              float64
	Y              float64
	Angle          float64
	Length         float64
	Width          float64
	Damage         int
	DamageInterval time.Duration
	ExpiresAt      time.Duration
	LastDamage     map[uint64]time.Duration
	TargetID       uint64
	Tracking       bool
}

type Explosion struct {
	ID             uint64
	OwnerID        string
	SpellID        string
	X              float64
	Y              float64
	Radius         float64
	Damage         int
	DamageInterval time.Duration
	ExpiresAt      time.Duration
	LastDamage     map[uint64]time.Duration
	FollowOwner    bool
}

type Meteor struct {
	ID              uint64
	X               float64
	Y               float64
	Radius          float64
	Damage          int
	ImpactAt        time.Duration
	ExpiresAt       time.Duration
	DamageInterval  time.Duration
	LastDamage      map[string]time.Duration
	OwnerID         string
	TargetsMonsters bool
	HitMonsters     map[uint64]struct{}
}

type activeMeteorShower struct {
	Definition MeteorShowerDefinition
	EndsAt     time.Duration
	Budget     float64
}

type Pickup struct {
	ID       uint64
	Kind     string
	X        float64
	Y        float64
	Value    int
	SpellIDs []string
}

type ProjectileRemoval struct {
	ID     uint64
	Reason string
}

type PlayerUpgradeOffer struct {
	Source        string
	Choices       []protocol.UpgradeChoice
	SelectedIndex int
}

type UpgradePhase struct {
	ID        uint64
	Source    string
	Deadline  time.Time
	Offers    map[string]*PlayerUpgradeOffer
	SpellPool []string
}

type Events struct {
	SpawnedProjectiles   []protocol.ProjectileSpawnedPayload
	RemovedProjectiles   []ProjectileRemoval
	DamageApplied        []protocol.DamageAppliedResult
	AppliedUpgrades      []protocol.UpgradeAppliedPayload
	MonsterAttacks       []protocol.MonsterAttackedPayload
	UpgradeOffersChanged bool
	MatchEnded           *protocol.MatchEndedPayload
	Metrics              TickMetrics
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
	StartedAt               time.Time
	Tick                    uint64
	Players                 map[string]*Player
	Monsters                map[uint64]*Monster
	Projectiles             map[uint64]*Projectile
	Beams                   map[uint64]*Beam
	Explosions              map[uint64]*Explosion
	Meteors                 map[uint64]*Meteor
	Pickups                 map[uint64]*Pickup
	TeamLevel               int
	TeamExperience          int
	TotalKills              int
	TeamLives               int
	EnemyScore              int
	TreasureEveryKills      int
	KillsSinceTreasure      int
	Elapsed                 time.Duration
	Finished                bool
	Level                   LevelDefinition
	activeSpawn             SpawnRateDefinition
	monsterHealthMultiplier float64
	monsterSpeedMultiplier  float64
	endingBossIDs           map[uint64]struct{}
	bossMonsterIDs          map[uint64]struct{}
	nextLevelEvent          int
	spawnBudget             float64
	nextMonsterID           uint64
	nextProjectileID        uint64
	nextBeamID              uint64
	nextExplosionID         uint64
	nextMeteorID            uint64
	nextPickupID            uint64
	rng                     *rand.Rand
	meteorShowers           []activeMeteorShower
	UpgradePhase            *UpgradePhase
	nextUpgradeOfferID      uint64
}

func NewMatch(startedAt time.Time, seed int64, levels ...LevelDefinition) *Match {
	level := levelOne
	if len(levels) > 0 {
		level = levels[0]
	}
	match := &Match{
		StartedAt:               startedAt,
		Players:                 make(map[string]*Player),
		Monsters:                make(map[uint64]*Monster),
		Projectiles:             make(map[uint64]*Projectile),
		Beams:                   make(map[uint64]*Beam),
		Explosions:              make(map[uint64]*Explosion),
		Meteors:                 make(map[uint64]*Meteor),
		Pickups:                 make(map[uint64]*Pickup),
		TeamLevel:               1,
		TreasureEveryKills:      PowerCrateEveryKills,
		monsterHealthMultiplier: 1,
		monsterSpeedMultiplier:  1,
		endingBossIDs:           make(map[uint64]struct{}),
		bossMonsterIDs:          make(map[uint64]struct{}),
		Level:                   level,
		rng:                     rand.New(rand.NewSource(seed)),
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
	spell, _ := SpellByID(character.DefaultSpellID)
	order := len(m.Players)
	angle := float64(order) * (2 * math.Pi / 6)
	player := &Player{
		ID:                 id,
		DisplayName:        displayName,
		CharacterID:        character.ID,
		SpellIDs:           append([]string(nil), character.StartingSpellIDs...),
		SpellLevels:        make(map[string]int, len(character.StartingSpellIDs)),
		LastSpellAttackAt:  make(map[string]time.Duration, len(character.StartingSpellIDs)),
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
		SpellKind:          spell.Kind,
		BeamLength:         spell.BeamLength,
		BeamWidth:          spell.BeamWidth,
		SpellDuration:      spell.Duration,
		DamageInterval:     spell.DamageInterval,
		ExplosionRadius:    spell.ExplosionRadius,
		ExplosionDuration:  spell.ExplosionDuration,
		ImpactDamage:       spell.ImpactDamage,
		SpellCooldown:      spell.Cooldown,
		SpellRange:         spell.Range,
		ProjectileRadius:   spell.Radius,
		BaseMaxHP:          character.MaxHP, BaseArmorPercent: character.ArmorPercent, BaseMovementSpeed: character.MovementSpeed,
		BaseHealthRegeneration: character.HealthRegeneration, BaseAttackBuffPercent: character.AttackBuffPercent, BaseCooldownPercent: character.CooldownPercent,
		BaseSpellDamage: spell.Damage, BaseProjectileSpeed: spell.ProjectileSpeed, BaseSpellBurst: spell.Burst, BaseSpellDirections: spell.Directions,
		ResurrectionDuration: character.ResurrectionDuration, ResurrectionRadius: character.ResurrectionRadius, ResurrectionImmunityDuration: character.ResurrectionImmunityDuration,
		Alive:        true,
		LastInputAt:  now,
		LastAttackAt: -spell.Cooldown,
	}
	for _, spellID := range player.SpellIDs {
		player.SpellLevels[spellID] = 1
		ownedSpell, _ := SpellByID(spellID)
		player.LastSpellAttackAt[spellID] = -ownedSpell.Cooldown
	}
	m.Players[id] = player
	m.TeamLives = min(MaximumTeamLives, m.TeamLives+1)
	m.assignAvailableLives()
	if m.UpgradePhase != nil {
		m.UpgradePhase.Offers[id] = m.createPlayerOffer(player, m.UpgradePhase.Source, m.UpgradePhase.SpellPool)
	}
	return player
}

func (m *Match) applySpellChoice(player *Player, spellID string) protocol.UpgradeAppliedPayload {
	spell, ok := SpellByID(spellID)
	if !ok || player.SpellLevels[spellID] > 0 || len(player.SpellIDs) >= maximumPlayerSpells {
		return protocol.UpgradeAppliedPayload{PlayerID: player.ID, Source: "spell_chest", Attribute: "spell:" + spellID}
	}
	player.SpellIDs = append(player.SpellIDs, spellID)
	player.SpellLevels[spellID] = 1
	player.LastSpellAttackAt[spellID] = m.Elapsed - spell.Cooldown
	return protocol.UpgradeAppliedPayload{PlayerID: player.ID, Source: "spell_chest", Attribute: "spell:" + spellID, AddedValue: 1, FinalValue: 1}
}

func resolveSpellLevel(spell SpellDefinition, level int) SpellDefinition {
	resolved := spell
	for currentLevel := 2; currentLevel <= level; currentLevel++ {
		for _, modifier := range spell.Levels[currentLevel] {
			switch modifier.Attribute {
			case "damage":
				resolved.Damage += int(math.Round(modifier.Value))
			case "cooldown_ms":
				resolved.Cooldown += time.Duration(modifier.Value * float64(time.Millisecond))
			case "projectile_speed":
				resolved.ProjectileSpeed += modifier.Value
			case "burst":
				resolved.Burst += int(math.Round(modifier.Value))
			case "directions":
				resolved.Directions += int(math.Round(modifier.Value))
			case "beam_length":
				resolved.BeamLength += modifier.Value
				resolved.Range = resolved.BeamLength
			case "beam_width":
				resolved.BeamWidth += modifier.Value
			case "linger_duration_ms":
				resolved.Duration += time.Duration(modifier.Value * float64(time.Millisecond))
				resolved.ExplosionDuration += time.Duration(modifier.Value * float64(time.Millisecond))
			case "damage_interval_ms":
				resolved.DamageInterval += time.Duration(modifier.Value * float64(time.Millisecond))
			case "explosion_radius":
				resolved.ExplosionRadius += modifier.Value
			}
		}
	}
	if resolved.Kind == "aura" {
		resolved.Range = resolved.ExplosionRadius
	}
	return resolved
}

func (player *Player) applyLegacySpellStats(spell SpellDefinition) {
	player.SpellDamage = spell.Damage
	player.ProjectileSpeed = spell.ProjectileSpeed
	player.SpellBurst = spell.Burst
	player.SpellDirections = spell.Directions
	player.SpellKind = spell.Kind
	player.BeamLength = spell.BeamLength
	player.BeamWidth = spell.BeamWidth
	player.SpellDuration = spell.Duration
	player.DamageInterval = spell.DamageInterval
	player.ExplosionRadius = spell.ExplosionRadius
	player.ExplosionDuration = spell.ExplosionDuration
	player.ImpactDamage = spell.ImpactDamage
	player.SpellCooldown = spell.Cooldown
	player.SpellRange = spell.Range
	player.ProjectileRadius = spell.Radius
}

func (m *Match) nthNearestMonster(x, y, maximumDistance float64, index int) *Monster {
	type candidate struct {
		monster  *Monster
		distance float64
	}
	candidates := make([]candidate, 0, len(m.Monsters))
	for _, monster := range m.Monsters {
		distance := math.Hypot(monster.X-x, monster.Y-y)
		if distance <= maximumDistance {
			candidates = append(candidates, candidate{monster: monster, distance: distance})
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].distance == candidates[j].distance {
			return candidates[i].monster.ID < candidates[j].monster.ID
		}
		return candidates[i].distance < candidates[j].distance
	})
	if index < 0 || index >= len(candidates) {
		return nil
	}
	return candidates[index].monster
}

func (m *Match) RemovePlayer(id string) {
	delete(m.Players, id)
	if m.UpgradePhase != nil {
		delete(m.UpgradePhase.Offers, id)
		if len(m.UpgradePhase.Offers) == 0 {
			m.UpgradePhase = nil
		}
	}
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
	if m.UpgradePhase != nil {
		events := Events{}
		if !now.Before(m.UpgradePhase.Deadline) || m.upgradePhaseComplete() {
			m.resolveUpgradePhase(&events)
		}
		events.Metrics.Total = time.Since(tickStarted)
		return events
	}
	if m.beginEarnedLevelIfReady(now) {
		events := Events{UpgradeOffersChanged: true}
		events.Metrics.Total = time.Since(tickStarted)
		return events
	}
	m.Elapsed += TickDuration
	events := Events{}
	phaseStarted := time.Now()
	m.updatePlayers(now)
	m.updateResurrections()
	events.Metrics.Movement = time.Since(phaseStarted)
	phaseStarted = time.Now()
	m.updateWeapons(&events)
	events.Metrics.Weapons = time.Since(phaseStarted)
	m.updateBeams(&events, &events.Metrics)
	m.updateProjectiles(&events, &events.Metrics)
	m.updateExplosions(&events, &events.Metrics)
	m.updateMeteors(&events)
	if m.Finished {
		events.Metrics.Total = time.Since(tickStarted)
		return events
	}
	phaseStarted = time.Now()
	m.updateMonsters(&events)
	events.Metrics.EnemyAI = time.Since(phaseStarted)
	phaseStarted = time.Now()
	m.updatePickups(&events, now)
	events.Metrics.Pickups = time.Since(phaseStarted)
	if m.UpgradePhase != nil {
		events.Metrics.Total = time.Since(tickStarted)
		return events
	}
	phaseStarted = time.Now()
	m.spawnMonsters()
	events.Metrics.Spawning = time.Since(phaseStarted)

	m.processLevelEvents(&events)
	if !m.Finished && !m.hasLivingPlayer() && !m.hasSoloResurrection() {
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
		spellSnapshots := make([]protocol.SnapshotSpell, 0, len(player.SpellIDs))
		for _, spellID := range player.SpellIDs {
			spell, ok := SpellByID(spellID)
			if !ok {
				continue
			}
			spell = resolveSpellLevel(spell, player.SpellLevels[spellID])
			spellSnapshots = append(spellSnapshots, protocol.SnapshotSpell{ID: spell.ID, Kind: spell.Kind, Level: player.SpellLevels[spellID], MaxLevel: spell.MaxLevel, Damage: int(math.Round(float64(spell.Damage) * (1 + player.AttackBuffPercent))), CooldownMs: time.Duration(float64(spell.Cooldown) * (1 - player.CooldownPercent)).Milliseconds(), Range: spell.Range, ProjectileSpeed: spell.ProjectileSpeed, ProjectileRadius: spell.Radius, Burst: spell.Burst, Directions: spell.Directions, BeamLength: spell.BeamLength, BeamWidth: spell.BeamWidth, DurationMs: spell.Duration.Milliseconds(), DamageIntervalMs: spell.DamageInterval.Milliseconds(), ExplosionRadius: spell.ExplosionRadius, ImpactDamage: int(math.Round(float64(spell.ImpactDamage) * (1 + player.AttackBuffPercent)))})
		}
		players = append(players, protocol.SnapshotPlayer{
			ID:                             player.ID,
			DisplayName:                    player.DisplayName,
			CharacterID:                    player.CharacterID,
			X:                              player.X,
			Y:                              player.Y,
			VelocityX:                      player.VelocityX,
			VelocityY:                      player.VelocityY,
			MovementSpeed:                  player.MovementSpeed,
			ArmorPercent:                   player.ArmorPercent,
			HealthRegeneration:             player.HealthRegeneration,
			AttackBuffPercent:              player.AttackBuffPercent,
			CooldownPercent:                player.CooldownPercent,
			SpellDamage:                    player.SpellDamage,
			ProjectileSpeed:                player.ProjectileSpeed,
			SpellBurst:                     player.SpellBurst,
			SpellDirections:                player.SpellDirections,
			Facing:                         player.Facing,
			HP:                             player.HP,
			MaxHP:                          player.MaxHP,
			Alive:                          player.Alive,
			LastProcessedInput:             player.LastProcessedInput,
			Kills:                          player.Kills,
			ResurrectionDurationMs:         player.ResurrectionDuration.Milliseconds(),
			ResurrectionRadius:             player.ResurrectionRadius,
			ResurrectionImmunityDurationMs: player.ResurrectionImmunityDuration.Milliseconds(),
			ResurrectionProgress:           resurrectionProgress(player),
			ResurrectionPending:            player.ResurrectionPending,
			ImmunityRemainingMs:            max(int64(0), (player.ImmuneUntil - m.Elapsed).Milliseconds()),
			Spells:                         spellSnapshots,
		})
	}

	monsters := make([]protocol.SnapshotMonster, 0, len(m.Monsters))
	monsterIDs := sortedUint64Keys(m.Monsters)
	for _, id := range monsterIDs {
		monster := m.Monsters[id]
		_, isBoss := m.bossMonsterIDs[id]
		monsters = append(monsters, protocol.SnapshotMonster{ID: id, TypeID: monster.TypeID, X: monster.X, Y: monster.Y, HP: monster.HP, MaxHP: monster.MaxHP, IsBoss: isBoss})
	}
	beams := make([]protocol.SnapshotBeam, 0, len(m.Beams))
	beamIDs := sortedUint64Keys(m.Beams)
	for _, id := range beamIDs {
		beam := m.Beams[id]
		beams = append(beams, protocol.SnapshotBeam{ID: id, OwnerID: beam.OwnerID, SpellID: beam.SpellID, X: beam.X, Y: beam.Y, Angle: beam.Angle, Length: beam.Length, Width: beam.Width, RemainingMs: max(int64(0), (beam.ExpiresAt - m.Elapsed).Milliseconds())})
	}
	explosions := make([]protocol.SnapshotExplosion, 0, len(m.Explosions))
	for _, id := range sortedUint64Keys(m.Explosions) {
		explosion := m.Explosions[id]
		explosions = append(explosions, protocol.SnapshotExplosion{ID: id, OwnerID: explosion.OwnerID, SpellID: explosion.SpellID, X: explosion.X, Y: explosion.Y, Radius: explosion.Radius, RemainingMs: max(int64(0), (explosion.ExpiresAt - m.Elapsed).Milliseconds())})
	}
	meteors := make([]protocol.SnapshotMeteor, 0, len(m.Meteors))
	for _, id := range sortedUint64Keys(m.Meteors) {
		meteor := m.Meteors[id]
		meteors = append(meteors, protocol.SnapshotMeteor{ID: id, X: meteor.X, Y: meteor.Y, Radius: meteor.Radius, ImpactInMs: max(int64(0), (meteor.ImpactAt - m.Elapsed).Milliseconds()), RemainingMs: max(int64(0), (meteor.ExpiresAt - m.Elapsed).Milliseconds()), Friendly: meteor.TargetsMonsters})
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
		Beams:        beams,
		Explosions:   explosions,
		Meteors:      meteors,
		Pickups:      pickups,
		Team: protocol.SnapshotTeam{
			Level:              m.TeamLevel,
			Experience:         m.TeamExperience,
			ExperienceRequired: RequiredExperience(m.TeamLevel),
			TotalKills:         m.TotalKills,
			Lives:              m.TeamLives,
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
		if !player.Alive {
			continue
		}
		for _, spellID := range player.SpellIDs {
			spell, ok := SpellByID(spellID)
			if !ok {
				continue
			}
			spell = resolveSpellLevel(spell, player.SpellLevels[spellID])
			if spell.Kind == "aura" {
				m.ensurePlayerAura(player, spell)
				continue
			}
			if spellID == player.SpellID {
				spell.Damage = player.SpellDamage
				spell.ProjectileSpeed = player.ProjectileSpeed
				spell.Burst = player.SpellBurst
				spell.Directions = player.SpellDirections
				spell.BeamLength = player.BeamLength
				spell.BeamWidth = player.BeamWidth
				spell.Duration = player.SpellDuration
				spell.DamageInterval = player.DamageInterval
				spell.ExplosionRadius = player.ExplosionRadius
				spell.ExplosionDuration = player.ExplosionDuration
				spell.ImpactDamage = player.ImpactDamage
				spell.Cooldown = player.SpellCooldown
				spell.Range = player.SpellRange
				spell.Radius = player.ProjectileRadius
			}
			cooldown := time.Duration(float64(spell.Cooldown) * (1 - player.CooldownPercent))
			if m.Elapsed-player.LastSpellAttackAt[spellID] < cooldown || !m.castPlayerSpell(player, spell, events) {
				continue
			}
			player.LastSpellAttackAt[spellID] = m.Elapsed
			if spellID == player.SpellID {
				player.LastAttackAt = m.Elapsed
			}
		}
	}
}

func (m *Match) ensurePlayerAura(player *Player, spell SpellDefinition) {
	for _, explosion := range m.Explosions {
		if explosion.OwnerID != player.ID || explosion.SpellID != spell.ID {
			continue
		}
		explosion.Radius = spell.ExplosionRadius
		explosion.Damage = int(math.Round(float64(spell.Damage) * (1 + player.AttackBuffPercent)))
		explosion.DamageInterval = spell.DamageInterval
		explosion.ExpiresAt = m.Elapsed + 2*TickDuration
		return
	}
	m.nextExplosionID++
	m.Explosions[m.nextExplosionID] = &Explosion{ID: m.nextExplosionID, OwnerID: player.ID, SpellID: spell.ID, X: player.X, Y: player.Y, Radius: spell.ExplosionRadius, Damage: int(math.Round(float64(spell.Damage) * (1 + player.AttackBuffPercent))), DamageInterval: spell.DamageInterval, ExpiresAt: m.Elapsed + 2*TickDuration, LastDamage: make(map[uint64]time.Duration), FollowOwner: true}
}

func (m *Match) castPlayerSpell(player *Player, spell SpellDefinition, events *Events) bool {
	var target *Monster
	nearest := spell.Range
	for _, monster := range m.Monsters {
		distance := math.Hypot(monster.X-player.X, monster.Y-player.Y)
		if distance <= nearest {
			nearest = distance
			target = monster
		}
	}
	if target == nil {
		return false
	}
	baseAngle := math.Atan2(target.Y-player.Y, target.X-player.X)
	spread := ProjectileSpread * math.Pi / 180
	if spell.Kind == "player_meteor" {
		for direction := range spell.Directions {
			meteorTarget := m.nthNearestMonster(player.X, player.Y, spell.Range, direction)
			if meteorTarget == nil {
				continue
			}
			m.nextMeteorID++
			impactAt := m.Elapsed + spell.Duration
			m.Meteors[m.nextMeteorID] = &Meteor{ID: m.nextMeteorID, X: meteorTarget.X, Y: meteorTarget.Y, Radius: spell.ExplosionRadius, Damage: int(math.Round(float64(spell.Damage) * (1 + player.AttackBuffPercent))), ImpactAt: impactAt, ExpiresAt: impactAt + 350*time.Millisecond, OwnerID: player.ID, TargetsMonsters: true, HitMonsters: make(map[uint64]struct{}), LastDamage: make(map[string]time.Duration)}
		}
		return true
	}
	if spell.Kind == "beam" || spell.Kind == "tracking_beam" {
		for direction := range spell.Directions {
			beamTarget := target
			if spell.Kind == "tracking_beam" {
				beamTarget = m.nthNearestMonster(player.X, player.Y, spell.Range, direction)
				if beamTarget == nil {
					continue
				}
			}
			angle := math.Atan2(beamTarget.Y-player.Y, beamTarget.X-player.X) + (float64(direction)-float64(spell.Directions-1)/2)*spread
			m.nextBeamID++
			m.Beams[m.nextBeamID] = &Beam{ID: m.nextBeamID, OwnerID: player.ID, SpellID: spell.ID, X: player.X, Y: player.Y, Angle: angle, Length: spell.BeamLength, Width: spell.BeamWidth, Damage: int(math.Round(float64(spell.Damage) * (1 + player.AttackBuffPercent))), DamageInterval: spell.DamageInterval, ExpiresAt: m.Elapsed + spell.Duration, LastDamage: make(map[uint64]time.Duration), TargetID: beamTarget.ID, Tracking: spell.Kind == "tracking_beam"}
		}
		return true
	}
	for direction := range spell.Directions {
		trajectory := baseAngle + (float64(direction)-float64(spell.Directions-1)/2)*spread
		for burst := range spell.Burst {
			angle := trajectory + (float64(burst)-float64(spell.Burst-1)/2)*(3*math.Pi/180)
			m.nextProjectileID++
			projectile := &Projectile{
				ID:                m.nextProjectileID,
				OwnerID:           player.ID,
				X:                 player.X,
				Y:                 player.Y,
				VelocityX:         math.Cos(angle) * spell.ProjectileSpeed,
				VelocityY:         math.Sin(angle) * spell.ProjectileSpeed,
				Damage:            int(math.Round(float64(spell.Damage) * (1 + player.AttackBuffPercent))),
				Range:             spell.Range,
				Radius:            spell.Radius,
				SpellID:           spell.ID,
				ExplosionRadius:   spell.ExplosionRadius,
				ExplosionDuration: spell.ExplosionDuration,
				DamageInterval:    spell.DamageInterval,
				ImpactDamage:      int(math.Round(float64(spell.ImpactDamage) * (1 + player.AttackBuffPercent))),
			}
			m.Projectiles[projectile.ID] = projectile
			events.SpawnedProjectiles = append(events.SpawnedProjectiles, protocol.ProjectileSpawnedPayload{
				ProjectileID: projectile.ID,
				OwnerID:      player.ID,
				WeaponID:     spell.ID,
				X:            projectile.X,
				Y:            projectile.Y,
				VelocityX:    projectile.VelocityX,
				VelocityY:    projectile.VelocityY,
				SpawnTick:    m.Tick,
			})
		}
	}
	return true
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
			m.createExplosion(projectile)
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
			m.createExplosion(projectile)
			m.removeProjectile(id, "obstacle_hit", events)
			continue
		}
		if projectile.TargetsPlayers {
			for _, playerID := range sortedPlayerIDs(m.Players) {
				player := m.Players[playerID]
				if !player.Alive || !overlaps(projectile.X, projectile.Y, projectile.Radius, player.X, player.Y, PlayerRadius) {
					continue
				}
				damage := max(1, int(math.Round(float64(projectile.Damage)*(1-player.ArmorPercent))))
				m.applyPlayerDamage(player, damage)
				metrics.ConfirmedHits++
				m.removeProjectile(id, "player_hit", events)
				break
			}
			continue
		}
		for monsterID, monster := range m.Monsters {
			metrics.CandidatePairs++
			metrics.NarrowChecks++
			if !overlaps(projectile.X, projectile.Y, projectile.Radius, monster.X, monster.Y, monster.Radius) {
				continue
			}
			if projectile.ExplosionRadius > 0 {
				m.applyMonsterDamage(monsterID, monster, projectile.OwnerID, projectile.ImpactDamage, events)
				m.createExplosion(projectile)
			} else {
				m.applyMonsterDamage(monsterID, monster, projectile.OwnerID, projectile.Damage, events)
			}
			metrics.ConfirmedHits++
			m.removeProjectile(id, "enemy_hit", events)
			if monster.HP <= 0 {
				m.killMonster(monsterID, projectile.OwnerID, events)
			}
			break
		}
	}
	metrics.NarrowPhase = time.Since(narrowStarted)
}

func (m *Match) createExplosion(projectile *Projectile) {
	if projectile.ExplosionRadius <= 0 {
		return
	}
	m.nextExplosionID++
	m.Explosions[m.nextExplosionID] = &Explosion{ID: m.nextExplosionID, OwnerID: projectile.OwnerID, SpellID: projectile.SpellID, X: projectile.X, Y: projectile.Y, Radius: projectile.ExplosionRadius, Damage: projectile.Damage, DamageInterval: projectile.DamageInterval, ExpiresAt: m.Elapsed + projectile.ExplosionDuration, LastDamage: make(map[uint64]time.Duration)}
}

func (m *Match) updateExplosions(events *Events, metrics *TickMetrics) {
	for id, explosion := range m.Explosions {
		if explosion.FollowOwner {
			if owner := m.Players[explosion.OwnerID]; owner != nil && owner.Alive {
				explosion.X, explosion.Y = owner.X, owner.Y
			}
		}
		for monsterID, monster := range m.Monsters {
			metrics.CandidatePairs++
			metrics.NarrowChecks++
			if !overlaps(explosion.X, explosion.Y, explosion.Radius, monster.X, monster.Y, monster.Radius) {
				continue
			}
			last, hitBefore := explosion.LastDamage[monsterID]
			if hitBefore && m.Elapsed-last < explosion.DamageInterval {
				continue
			}
			explosion.LastDamage[monsterID] = m.Elapsed
			m.applyMonsterDamage(monsterID, monster, explosion.OwnerID, explosion.Damage, events)
			metrics.ConfirmedHits++
			if monster.HP <= 0 {
				m.killMonster(monsterID, explosion.OwnerID, events)
			}
		}
		if m.Elapsed >= explosion.ExpiresAt {
			delete(m.Explosions, id)
		}
	}
}

func (m *Match) updateBeams(events *Events, metrics *TickMetrics) {
	for id, beam := range m.Beams {
		if beam.Tracking {
			owner := m.Players[beam.OwnerID]
			if owner == nil || !owner.Alive {
				delete(m.Beams, id)
				continue
			}
			target := m.Monsters[beam.TargetID]
			if target == nil {
				target = m.nthNearestMonster(owner.X, owner.Y, beam.Length, 0)
				if target == nil {
					delete(m.Beams, id)
					continue
				}
				beam.TargetID = target.ID
			}
			beam.X, beam.Y = owner.X, owner.Y
			beam.Angle = math.Atan2(target.Y-owner.Y, target.X-owner.X)
		}
		endX := beam.X + math.Cos(beam.Angle)*beam.Length
		endY := beam.Y + math.Sin(beam.Angle)*beam.Length
		for monsterID, monster := range m.Monsters {
			metrics.CandidatePairs++
			metrics.NarrowChecks++
			if distanceToSegment(monster.X, monster.Y, beam.X, beam.Y, endX, endY) > monster.Radius+beam.Width/2 {
				continue
			}
			last, hitBefore := beam.LastDamage[monsterID]
			if hitBefore && m.Elapsed-last < beam.DamageInterval {
				continue
			}
			beam.LastDamage[monsterID] = m.Elapsed
			m.applyMonsterDamage(monsterID, monster, beam.OwnerID, beam.Damage, events)
			metrics.ConfirmedHits++
			if monster.HP <= 0 {
				m.killMonster(monsterID, beam.OwnerID, events)
			}
		}
		if m.Elapsed >= beam.ExpiresAt {
			delete(m.Beams, id)
		}
	}
}

func (m *Match) updateMonsters(events *Events) {
	for _, monster := range m.Monsters {
		target := m.nearestLivingPlayer(monster.X, monster.Y)
		if target == nil {
			return
		}
		m.moveMonster(monster, target)
		m.castMonsterSpell(monster, target, events)

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
			m.applyPlayerDamage(player, damage)
		}
	}
	m.separateMonsters()
}

func (m *Match) moveMonster(monster *Monster, target *Player) {
	dx := target.X - monster.X
	dy := target.Y - monster.Y
	distance := math.Hypot(dx, dy)
	speed := monster.Speed
	dirX, dirY := 0.0, 0.0
	if distance > 0 {
		dirX, dirY = dx/distance, dy/distance
	}
	if monster.MovementMode == "dash" {
		speed, dirX, dirY = m.updateMonsterDash(monster, target, distance, dirX, dirY)
	}
	if dirX == 0 && dirY == 0 {
		return
	}
	step := speed * TickDuration.Seconds()
	attemptX := monster.X + dirX*step
	attemptY := monster.Y + dirY*step
	resolvedX, resolvedY := resolveWorldAndObstacles(attemptX, attemptY, monster.Radius, m.Level.Obstacles)
	if resolvedX != attemptX || resolvedY != attemptY {
		resolvedX += -dirY * step * 0.55
		resolvedY += dirX * step * 0.55
		resolvedX, resolvedY = resolveWorldAndObstacles(resolvedX, resolvedY, monster.Radius, m.Level.Obstacles)
	}
	monster.X = resolvedX
	monster.Y = resolvedY
}

func (m *Match) updateMonsterDash(monster *Monster, target *Player, distance, dirX, dirY float64) (float64, float64, float64) {
	switch monster.DashState {
	case "windup":
		monster.DashTimer += TickDuration
		speed := monster.Speed * MonsterDashWindupSpeedFactor
		if monster.DashTimer >= monster.DashWindup {
			monster.DashState = "dash"
			monster.DashTimer = 0
			if distance > 0 {
				monster.DashDirX, monster.DashDirY = dirX, dirY
			} else {
				monster.DashDirX, monster.DashDirY = 1, 0
			}
		}
		return speed, dirX, dirY
	case "dash":
		monster.DashTimer += TickDuration
		speed := monster.Speed * monster.DashSpeedMultiplier
		if monster.DashTimer >= monster.DashDuration {
			monster.DashState = "idle"
			monster.DashTimer = 0
		}
		return speed, monster.DashDirX, monster.DashDirY
	default:
		monster.DashTimer += TickDuration
		if monster.DashTimer >= monster.DashInterval {
			monster.DashState = "windup"
			monster.DashTimer = 0
		}
		return monster.Speed, dirX, dirY
	}
}

func (m *Match) castMonsterSpell(monster *Monster, target *Player, events *Events) {
	if len(monster.SpellIDs) == 0 {
		return
	}
	if monster.LastSpellAt == nil {
		monster.LastSpellAt = make(map[string]time.Duration, len(monster.SpellIDs))
	}
	dx, dy := target.X-monster.X, target.Y-monster.Y
	distance := math.Hypot(dx, dy)
	if distance <= 0 {
		return
	}
	baseAngle := math.Atan2(dy, dx)
	spread := ProjectileSpread * math.Pi / 180
	for _, spellID := range monster.SpellIDs {
		spell, ok := SpellByID(spellID)
		if !ok || distance > spell.Range || m.Elapsed-monster.LastSpellAt[spellID] < spell.Cooldown {
			continue
		}
		if spell.Kind == "melee" {
			monster.LastSpellAt[spellID] = m.Elapsed
			events.MonsterAttacks = append(events.MonsterAttacks, protocol.MonsterAttackedPayload{MonsterID: monster.ID, SpellID: spell.ID, TargetPlayerID: target.ID})
			damage := max(1, int(math.Round(float64(spell.Damage)*monster.AttackDamageMultiplier*(1-target.ArmorPercent))))
			m.applyPlayerDamage(target, damage)
			continue
		}
		if spell.Kind != "projectile" {
			continue
		}
		monster.LastSpellAt[spellID] = m.Elapsed
		for direction := range spell.Directions {
			angle := baseAngle + (float64(direction)-float64(spell.Directions-1)/2)*spread
			m.nextProjectileID++
			projectile := &Projectile{ID: m.nextProjectileID, OwnerID: "enemy:" + strconv.FormatUint(monster.ID, 10), X: monster.X, Y: monster.Y, VelocityX: math.Cos(angle) * spell.ProjectileSpeed, VelocityY: math.Sin(angle) * spell.ProjectileSpeed, Damage: max(1, int(math.Round(float64(spell.Damage)*monster.AttackDamageMultiplier))), Range: spell.Range, Radius: spell.Radius, SpellID: spell.ID, TargetsPlayers: true}
			m.Projectiles[projectile.ID] = projectile
			events.SpawnedProjectiles = append(events.SpawnedProjectiles, protocol.ProjectileSpawnedPayload{ProjectileID: projectile.ID, OwnerID: projectile.OwnerID, WeaponID: spell.ID, X: projectile.X, Y: projectile.Y, VelocityX: projectile.VelocityX, VelocityY: projectile.VelocityY, SpawnTick: m.Tick})
		}
	}
}

type monsterGridCell struct{ X, Y int }

func (m *Match) separateMonsters() {
	if len(m.Monsters) < 2 {
		return
	}
	ids := sortedUint64Keys(m.Monsters)
	grid := make(map[monsterGridCell][]uint64, len(ids))
	for _, id := range ids {
		monster := m.Monsters[id]
		cell := monsterGridCell{X: int(math.Floor(monster.X / MonsterSeparationCellSize)), Y: int(math.Floor(monster.Y / MonsterSeparationCellSize))}
		grid[cell] = append(grid[cell], id)
	}
	deltaX := make(map[uint64]float64, len(ids))
	deltaY := make(map[uint64]float64, len(ids))
	for _, id := range ids {
		monster := m.Monsters[id]
		cell := monsterGridCell{X: int(math.Floor(monster.X / MonsterSeparationCellSize)), Y: int(math.Floor(monster.Y / MonsterSeparationCellSize))}
		for offsetY := -1; offsetY <= 1; offsetY++ {
			for offsetX := -1; offsetX <= 1; offsetX++ {
				for _, otherID := range grid[monsterGridCell{X: cell.X + offsetX, Y: cell.Y + offsetY}] {
					if otherID <= id {
						continue
					}
					other := m.Monsters[otherID]
					dx, dy := other.X-monster.X, other.Y-monster.Y
					distance := math.Hypot(dx, dy)
					desired := (monster.Radius + other.Radius) * MonsterSeparationRadiusFactor
					if distance >= desired {
						continue
					}
					if distance < 0.001 {
						angle := float64((id*31+otherID*17)%360) * math.Pi / 180
						dx, dy, distance = math.Cos(angle), math.Sin(angle), 1
					}
					push := min(MonsterSeparationMaxStep, (desired-distance)*MonsterSeparationStrength*0.5)
					normalX, normalY := dx/distance, dy/distance
					deltaX[id] -= normalX * push
					deltaY[id] -= normalY * push
					deltaX[otherID] += normalX * push
					deltaY[otherID] += normalY * push
				}
			}
		}
	}
	for _, id := range ids {
		monster := m.Monsters[id]
		dx, dy := deltaX[id], deltaY[id]
		length := math.Hypot(dx, dy)
		if length > MonsterSeparationMaxStep {
			dx *= MonsterSeparationMaxStep / length
			dy *= MonsterSeparationMaxStep / length
		}
		monster.X, monster.Y = resolveWorldAndObstacles(monster.X+dx, monster.Y+dy, monster.Radius, m.Level.Obstacles)
	}
}

func (m *Match) updatePickups(events *Events, now time.Time) {
	for pickupID, pickup := range m.Pickups {
		if pickup.Kind == "power_crate" || pickup.Kind == "spell_chest" {
			if m.nearestLivingPlayerWithin(pickup.X, pickup.Y, PowerCrateRadius) == nil {
				continue
			}
			delete(m.Pickups, pickupID)
			source := "treasure_chest"
			if pickup.Kind == "spell_chest" {
				source = "spell_chest"
			}
			m.beginUpgradePhase(now, source, pickup.SpellIDs)
			events.UpgradeOffersChanged = true
			return
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
		if m.beginEarnedLevelIfReady(now) {
			events.UpgradeOffersChanged = true
			return
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

func (m *Match) spawnMonsterOf(enemyID string, requestedMultipliers ...*EnemyStatMultipliers) uint64 {
	definition, ok := EnemyByID(enemyID)
	if !ok {
		return 0
	}
	living := m.livingPlayers()
	if len(living) == 0 {
		return 0
	}
	multipliers := &EnemyStatMultipliers{}
	if len(requestedMultipliers) > 0 && requestedMultipliers[0] != nil {
		multipliers = requestedMultipliers[0]
	}
	healthMultiplier := positiveMultiplier(multipliers.Health)
	speedMultiplier := positiveMultiplier(multipliers.MovementSpeed)
	damageMultiplier := positiveMultiplier(multipliers.AttackDamage)
	radiusMultiplier := positiveMultiplier(multipliers.CollisionRadius)
	cooldownMultiplier := positiveMultiplier(multipliers.ContactCooldown)
	experienceMultiplier := positiveMultiplier(multipliers.ExperienceDrop)
	scoreMultiplier := positiveMultiplier(multipliers.Score)
	radius := definition.Radius * radiusMultiplier
	target := living[m.rng.Intn(len(living))]
	for range 10 {
		angle := m.rng.Float64() * 2 * math.Pi
		distance := 700 + m.rng.Float64()*200
		x := clamp(target.X+math.Cos(angle)*distance, radius, WorldWidth-radius)
		y := clamp(target.Y+math.Sin(angle)*distance, radius, WorldHeight-radius)
		if collidesObstacle(x, y, radius, m.Level.Obstacles) {
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
		maxHP := max(1, int(math.Round(float64(definition.MaxHP)*m.monsterHealthMultiplier*healthMultiplier)))
		m.Monsters[m.nextMonsterID] = &Monster{ID: m.nextMonsterID, TypeID: definition.ID, X: x, Y: y, HP: maxHP, MaxHP: maxHP, Speed: definition.Speed * m.monsterSpeedMultiplier * speedMultiplier, Radius: radius, ContactDamage: max(1, int(math.Round(float64(definition.ContactDamage)*damageMultiplier))), Armor: definition.Armor, SpellIDs: append([]string(nil), definition.SpellIDs...), LastSpellAt: make(map[string]time.Duration, len(definition.SpellIDs)), AttackDamageMultiplier: damageMultiplier, ContactDelay: time.Duration(float64(definition.ContactDelay) * cooldownMultiplier), Experience: max(1, int(math.Round(float64(definition.Experience)*experienceMultiplier))), Score: max(1, int(math.Round(float64(definition.Score)*scoreMultiplier))), LastContact: make(map[string]time.Duration), MovementMode: definition.Movement.Mode, DashInterval: definition.Movement.DashInterval, DashWindup: definition.Movement.DashWindup, DashDuration: definition.Movement.DashDuration, DashSpeedMultiplier: definition.Movement.DashSpeedMultiplier}
		return m.nextMonsterID
	}
	return 0
}

func positiveMultiplier(value float64) float64 {
	if value > 0 {
		return value
	}
	return 1
}

func enemyDamageAfterArmor(monster *Monster, damage int) int {
	return max(1, damage-monster.Armor)
}

func (m *Match) applyMonsterDamage(monsterID uint64, monster *Monster, attackerID string, rawDamage int, events *Events) {
	damage := enemyDamageAfterArmor(monster, rawDamage)
	monster.HP = max(0, monster.HP-damage)
	events.DamageApplied = append(events.DamageApplied, protocol.DamageAppliedResult{
		AttackerID:  attackerID,
		TargetType:  "monster",
		TargetID:    strconv.FormatUint(monsterID, 10),
		Amount:      damage,
		RemainingHP: monster.HP,
		Death:       monster.HP == 0,
	})
}

func (m *Match) removeProjectile(id uint64, reason string, events *Events) {
	delete(m.Projectiles, id)
	events.RemovedProjectiles = append(events.RemovedProjectiles, ProjectileRemoval{ID: id, Reason: reason})
}

func (m *Match) killMonster(monsterID uint64, ownerID string, events *Events) {
	monster := m.Monsters[monsterID]
	_, wasBoss := m.bossMonsterIDs[monsterID]
	_, endsMatch := m.endingBossIDs[monsterID]
	delete(m.Monsters, monsterID)
	delete(m.bossMonsterIDs, monsterID)
	m.nextPickupID++
	m.Pickups[m.nextPickupID] = &Pickup{ID: m.nextPickupID, Kind: "experience", X: monster.X, Y: monster.Y, Value: monster.Experience}
	m.TotalKills++
	m.KillsSinceTreasure++
	m.EnemyScore += monster.Score
	cadenceDrop := m.TreasureEveryKills > 0 && m.KillsSinceTreasure >= m.TreasureEveryKills
	if cadenceDrop {
		m.KillsSinceTreasure = 0
	}
	if (wasBoss && !endsMatch) || cadenceDrop {
		m.nextPickupID++
		x, y := resolveWorldAndObstacles(monster.X+44, monster.Y, 18, m.Level.Obstacles)
		m.Pickups[m.nextPickupID] = &Pickup{ID: m.nextPickupID, Kind: "power_crate", X: x, Y: y}
	}
	if owner, ok := m.Players[ownerID]; ok {
		owner.Kills++
	}
	if endsMatch {
		delete(m.endingBossIDs, monsterID)
		m.finish("won", events)
	}
}

func availableUpgradeAttributes(player *Player) []string {
	available := []string{"max_health", "health_regeneration", "attack_buff"}
	if player.ArmorPercent < MaximumArmorPercent {
		available = append(available, "armor")
	}
	if player.MovementSpeed < player.BaseMovementSpeed*(1+MaximumMovementBonus) {
		available = append(available, "movement_speed")
	}
	if player.CooldownPercent < 0.60 {
		available = append(available, "cooldown")
	}
	for _, spellID := range player.SpellIDs {
		spell, ok := SpellByID(spellID)
		if ok && player.SpellLevels[spellID] < spell.MaxLevel {
			available = append(available, "spell:"+spellID+":level")
		}
	}
	return available
}

func (m *Match) applyUpgrade(player *Player, source, attribute string) protocol.UpgradeAppliedPayload {
	if spellID, ok := spellLevelAttribute(attribute); ok {
		spell, exists := SpellByID(spellID)
		current := player.SpellLevels[spellID]
		if !exists || current < 1 || current >= spell.MaxLevel {
			return protocol.UpgradeAppliedPayload{PlayerID: player.ID, Source: source, Attribute: attribute, BaseValue: float64(current), FinalValue: float64(current)}
		}
		player.SpellLevels[spellID] = current + 1
		if spellID == player.SpellID {
			player.applyLegacySpellStats(resolveSpellLevel(spell, current+1))
		}
		return protocol.UpgradeAppliedPayload{PlayerID: player.ID, Source: source, Attribute: attribute, BaseValue: float64(current), AddedValue: 1, FinalValue: float64(current + 1)}
	}
	baseValue, addedValue, finalValue := 0.0, 0.0, 0.0
	switch attribute {
	case "max_health":
		baseValue, addedValue = float64(player.BaseMaxHP), 20
		player.MaxHP += 20
		player.HP += 20
		finalValue = float64(player.MaxHP)
	case "armor":
		baseValue = player.BaseArmorPercent
		before := player.ArmorPercent
		player.ArmorPercent = min(MaximumArmorPercent, player.ArmorPercent+0.05)
		addedValue = player.ArmorPercent - before
		finalValue = player.ArmorPercent
	case "movement_speed":
		baseValue = player.BaseMovementSpeed
		before := player.MovementSpeed
		player.MovementSpeed = min(player.BaseMovementSpeed*(1+MaximumMovementBonus), player.MovementSpeed+player.BaseMovementSpeed*0.08)
		addedValue = player.MovementSpeed - before
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
		baseValue = player.BaseCooldownPercent
		before := player.CooldownPercent
		player.CooldownPercent = min(0.60, player.CooldownPercent+0.08)
		addedValue = player.CooldownPercent - before
		finalValue = player.CooldownPercent
	case "spell_damage":
		increment := 4
		if player.SpellKind == "beam" {
			increment = 6
		} else if player.SpellKind == "explosive_projectile" {
			increment = 8
		}
		baseValue, addedValue = float64(player.BaseSpellDamage), float64(increment)
		player.SpellDamage += increment
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
	case "beam_length":
		spell, _ := SpellByID(player.SpellID)
		baseValue, addedValue = spell.BeamLength, 100
		player.BeamLength += 100
		player.SpellRange = player.BeamLength
		finalValue = player.BeamLength
	case "beam_width":
		spell, _ := SpellByID(player.SpellID)
		baseValue, addedValue = spell.BeamWidth, 10
		player.BeamWidth += 10
		finalValue = player.BeamWidth
	case "spell_duration":
		spell, _ := SpellByID(player.SpellID)
		baseValue, addedValue = float64(spell.Duration.Milliseconds()), 250
		player.SpellDuration += 250 * time.Millisecond
		finalValue = float64(player.SpellDuration.Milliseconds())
	case "explosion_radius":
		spell, _ := SpellByID(player.SpellID)
		baseValue, addedValue = spell.ExplosionRadius, 20
		player.ExplosionRadius += 20
		finalValue = player.ExplosionRadius
	case "explosion_duration":
		spell, _ := SpellByID(player.SpellID)
		baseValue, addedValue = float64(spell.ExplosionDuration.Milliseconds()), 250
		player.ExplosionDuration += 250 * time.Millisecond
		finalValue = float64(player.ExplosionDuration.Milliseconds())
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
		if event.Type == "boss" && len(m.livingPlayers()) == 0 {
			return
		}
		m.nextLevelEvent++
		switch event.Type {
		case "spawn_rate":
			if event.SpawnRate != nil {
				m.activeSpawn = *event.SpawnRate
			}
		case "monster_buff":
			if event.MonsterBuff != nil {
				m.monsterHealthMultiplier *= event.MonsterBuff.HealthMultiplier
				m.monsterSpeedMultiplier *= event.MonsterBuff.SpeedMultiplier
				for _, monster := range m.Monsters {
					monster.HP = max(1, int(math.Round(float64(monster.HP)*event.MonsterBuff.HealthMultiplier)))
					monster.MaxHP = max(1, int(math.Round(float64(monster.MaxHP)*event.MonsterBuff.HealthMultiplier)))
					monster.Speed *= event.MonsterBuff.SpeedMultiplier
				}
			}
		case "meteor_shower":
			if event.MeteorShower != nil {
				m.meteorShowers = append(m.meteorShowers, activeMeteorShower{Definition: *event.MeteorShower, EndsAt: m.Elapsed + event.MeteorShower.Duration})
			}
		case "spell_chest":
			if event.SpellChest != nil {
				m.nextPickupID++
				m.Pickups[m.nextPickupID] = &Pickup{ID: m.nextPickupID, Kind: "spell_chest", X: event.SpellChest.X, Y: event.SpellChest.Y, SpellIDs: append([]string(nil), event.SpellChest.SpellIDs...)}
			}
		case "treasure_rate":
			if event.TreasureRate != nil {
				m.TreasureEveryKills = event.TreasureRate.KillsPerChest
				m.KillsSinceTreasure = min(m.KillsSinceTreasure, m.TreasureEveryKills-1)
			}
		case "boss":
			spawned := 0
			for range max(1, event.BossCount) {
				monsterID := m.spawnMonsterOf(event.EnemyID, event.BossMultipliers)
				if monsterID != 0 {
					spawned++
					m.bossMonsterIDs[monsterID] = struct{}{}
				}
				if monsterID != 0 && event.EndMatchOnDeath {
					m.endingBossIDs[monsterID] = struct{}{}
				}
			}
			if spawned == 0 {
				m.nextLevelEvent--
				return
			}
		case "end":
			m.finish("won", events)
		}
	}
}

func (m *Match) finish(outcome string, events *Events) {
	if m.Finished {
		return
	}
	m.Finished = true
	for id := range m.Projectiles {
		m.removeProjectile(id, "match_ended", events)
	}
	clear(m.Beams)
	clear(m.Explosions)
	clear(m.Meteors)
	m.meteorShowers = nil
	clear(m.endingBossIDs)
	clear(m.bossMonsterIDs)
	events.MatchEnded = &protocol.MatchEndedPayload{
		Outcome:    outcome,
		SurvivalMs: m.Elapsed.Milliseconds(),
		TeamLevel:  m.TeamLevel,
		TotalKills: m.TotalKills,
		Score:      m.EnemyScore + m.TeamLevel*250 + int(m.Elapsed/time.Second),
	}
}

func (m *Match) updateMeteors(events *Events) {
	activeShowers := m.meteorShowers[:0]
	for _, shower := range m.meteorShowers {
		if m.Elapsed >= shower.EndsAt {
			continue
		}
		shower.Budget += shower.Definition.RatePerSecond * TickDuration.Seconds()
		for shower.Budget >= 1 && len(m.Meteors) < 256 {
			shower.Budget--
			m.spawnMeteor(shower.Definition)
		}
		activeShowers = append(activeShowers, shower)
	}
	m.meteorShowers = activeShowers

	for id, meteor := range m.Meteors {
		if m.Elapsed >= meteor.ExpiresAt {
			delete(m.Meteors, id)
			continue
		}
		if m.Elapsed < meteor.ImpactAt {
			continue
		}
		if meteor.TargetsMonsters {
			for _, monsterID := range sortedUint64Keys(m.Monsters) {
				monster := m.Monsters[monsterID]
				if _, hit := meteor.HitMonsters[monsterID]; hit || math.Hypot(monster.X-meteor.X, monster.Y-meteor.Y) > meteor.Radius+monster.Radius {
					continue
				}
				meteor.HitMonsters[monsterID] = struct{}{}
				m.applyMonsterDamage(monsterID, monster, meteor.OwnerID, meteor.Damage, events)
				if monster.HP <= 0 {
					m.killMonster(monsterID, meteor.OwnerID, events)
				}
			}
			continue
		}
		for _, playerID := range sortedPlayerIDs(m.Players) {
			player := m.Players[playerID]
			if !player.Alive || math.Hypot(player.X-meteor.X, player.Y-meteor.Y) > meteor.Radius+PlayerRadius {
				continue
			}
			if last, hit := meteor.LastDamage[playerID]; hit && m.Elapsed-last < meteor.DamageInterval {
				continue
			}
			meteor.LastDamage[playerID] = m.Elapsed
			damage := max(1, int(math.Round(float64(meteor.Damage)*(1-player.ArmorPercent))))
			m.applyPlayerDamage(player, damage)
		}
	}
}

func (m *Match) spawnMeteor(definition MeteorShowerDefinition) {
	living := make([]*Player, 0, len(m.Players))
	for _, id := range sortedPlayerIDs(m.Players) {
		if m.Players[id].Alive {
			living = append(living, m.Players[id])
		}
	}
	if len(living) == 0 {
		return
	}
	target := living[m.rng.Intn(len(living))]
	angle, distance := m.rng.Float64()*2*math.Pi, m.rng.Float64()*420
	x := clamp(target.X+math.Cos(angle)*distance, definition.Radius, WorldWidth-definition.Radius)
	y := clamp(target.Y+math.Sin(angle)*distance, definition.Radius, WorldHeight-definition.Radius)
	linger := definition.LingerMin
	if spread := definition.LingerMax - definition.LingerMin; spread > 0 {
		linger += time.Duration(m.rng.Int63n(int64(spread) + 1))
	}
	m.nextMeteorID++
	impactAt := m.Elapsed + definition.Warning
	m.Meteors[m.nextMeteorID] = &Meteor{ID: m.nextMeteorID, X: x, Y: y, Radius: definition.Radius, Damage: definition.Damage, ImpactAt: impactAt, ExpiresAt: impactAt + linger, DamageInterval: definition.DamageInterval, LastDamage: make(map[string]time.Duration)}
}

func distanceToSegment(px, py, ax, ay, bx, by float64) float64 {
	dx, dy := bx-ax, by-ay
	lengthSquared := dx*dx + dy*dy
	if lengthSquared == 0 {
		return math.Hypot(px-ax, py-ay)
	}
	t := clamp(((px-ax)*dx+(py-ay)*dy)/lengthSquared, 0, 1)
	return math.Hypot(px-(ax+t*dx), py-(ay+t*dy))
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
