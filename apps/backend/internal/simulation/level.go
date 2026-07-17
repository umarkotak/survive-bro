package simulation

import (
	"sort"
	"time"
)

type SpellDefinition struct {
	ID                string
	Kind              string
	Damage            int
	Cooldown          time.Duration
	ProjectileSpeed   float64
	Range             float64
	Radius            float64
	Burst             int
	Directions        int
	BeamLength        float64
	BeamWidth         float64
	Duration          time.Duration
	DamageInterval    time.Duration
	ExplosionRadius   float64
	ExplosionDuration time.Duration
	ImpactDamage      int
	MaxLevel          int
	Levels            map[int][]SpellModifier
	PlayerAvailable   bool
}

type SpellModifier struct {
	Attribute string
	Value     float64
}

var maximumPlayerSpells = 4

type CharacterDefinition struct {
	ID                           string
	Name                         string
	SpriteID                     string
	MaxHP                        int
	ArmorPercent                 float64
	MovementSpeed                float64
	HealthRegeneration           float64
	AttackBuffPercent            float64
	CooldownPercent              float64
	PickupRadius                 float64
	ResurrectionDuration         time.Duration
	ResurrectionRadius           float64
	ResurrectionImmunityDuration time.Duration
	DefaultSpellID               string
	StartingSpellIDs             []string
}

type EnemyDefinition struct {
	ID            string
	Name          string
	SpriteID      string
	Score         int
	Experience    int
	MaxHP         int
	Speed         float64
	Radius        float64
	ContactDamage int
	Armor         int
	SpellIDs      []string
	ContactDelay  time.Duration
}

type SpawnEntry struct {
	EnemyID string
	Weight  int
}
type SpawnRateDefinition struct {
	RatePerSecond float64
	MaxLiving     int
	Entries       []SpawnEntry
}

type MonsterBuffDefinition struct {
	HealthMultiplier float64
	SpeedMultiplier  float64
}

type MeteorShowerDefinition struct {
	Duration       time.Duration
	RatePerSecond  float64
	Warning        time.Duration
	LingerMin      time.Duration
	LingerMax      time.Duration
	Radius         float64
	Damage         int
	DamageInterval time.Duration
}

type SpellChestDefinition struct {
	X        float64
	Y        float64
	SpellIDs []string
}

type TreasureRateDefinition struct {
	KillsPerChest int
}

type EnemyStatMultipliers struct {
	Health          float64
	MovementSpeed   float64
	AttackDamage    float64
	CollisionRadius float64
	ContactCooldown float64
	ExperienceDrop  float64
	Score           float64
}

type LevelEvent struct {
	ID              string
	At              time.Duration
	Type            string
	Title           string
	Description     string
	Show            bool
	SpawnRate       *SpawnRateDefinition
	MonsterBuff     *MonsterBuffDefinition
	MeteorShower    *MeteorShowerDefinition
	SpellChest      *SpellChestDefinition
	TreasureRate    *TreasureRateDefinition
	EnemyID         string
	EndMatchOnDeath bool
	BossMultipliers *EnemyStatMultipliers
	BossCount       int
}

type LevelDefinition struct {
	ID               string
	Name             string
	Duration         time.Duration
	TerrainAssetIDs  []string
	ObstacleAssetIDs []string
	Obstacles        []Obstacle
	Events           []LevelEvent
}

var spells = map[string]SpellDefinition{
	"fireball":         {ID: "fireball", Kind: "projectile", Damage: 20, Cooldown: 750 * time.Millisecond, ProjectileSpeed: 700, Range: 700, Radius: 10, Burst: 1, Directions: 1, MaxLevel: 5, PlayerAvailable: true},
	"enemy-slime-ball": {ID: "enemy-slime-ball", Kind: "projectile", Damage: 18, Cooldown: time.Second, ProjectileSpeed: 360, Range: 360, Radius: 12, Burst: 1, Directions: 1, MaxLevel: 1},
	"slime-punch":      {ID: "slime-punch", Kind: "melee", Damage: 14, Cooldown: 1200 * time.Millisecond, Range: 90, Burst: 1, Directions: 1, MaxLevel: 1},
	"soul-track":       {ID: "soul-track", Kind: "beam", Damage: 18, Cooldown: 1500 * time.Millisecond, Range: 520, Radius: 16, Burst: 1, Directions: 1, BeamLength: 520, BeamWidth: 32, Duration: time.Second, DamageInterval: 500 * time.Millisecond, MaxLevel: 7, PlayerAvailable: true},
	"rocket":           {ID: "rocket", Kind: "explosive_projectile", Damage: 30, ImpactDamage: 20, Cooldown: 1600 * time.Millisecond, ProjectileSpeed: 480, Range: 850, Radius: 12, Burst: 1, Directions: 1, ExplosionRadius: 80, ExplosionDuration: time.Second, DamageInterval: 500 * time.Millisecond, MaxLevel: 7, PlayerAvailable: true},
	"heavy-aura":       {ID: "heavy-aura", Kind: "aura", Damage: 4, Cooldown: 500 * time.Millisecond, Range: 110, ExplosionRadius: 110, Duration: 550 * time.Millisecond, DamageInterval: 500 * time.Millisecond, Burst: 1, Directions: 1, MaxLevel: 7, PlayerAvailable: true, Levels: map[int][]SpellModifier{2: {{Attribute: "explosion_radius", Value: 30}}, 3: {{Attribute: "damage", Value: 2}}, 4: {{Attribute: "damage_interval_ms", Value: -75}}, 5: {{Attribute: "explosion_radius", Value: 40}}, 6: {{Attribute: "damage", Value: 4}}, 7: {{Attribute: "damage_interval_ms", Value: -75}}}},
	"meteorite":        {ID: "meteorite", Kind: "player_meteor", Damage: 42, Cooldown: 3 * time.Second, Range: 700, ExplosionRadius: 85, Duration: 900 * time.Millisecond, Burst: 1, Directions: 1, MaxLevel: 8, PlayerAvailable: true, Levels: map[int][]SpellModifier{2: {{Attribute: "explosion_radius", Value: 20}}, 3: {{Attribute: "damage", Value: 16}}, 4: {{Attribute: "linger_duration_ms", Value: -200}}, 5: {{Attribute: "cooldown_ms", Value: -400}}, 6: {{Attribute: "explosion_radius", Value: 25}}, 7: {{Attribute: "damage", Value: 24}}, 8: {{Attribute: "directions", Value: 1}}}},
	"tracking-beam":    {ID: "tracking-beam", Kind: "tracking_beam", Damage: 6, Cooldown: 2400 * time.Millisecond, Range: 600, BeamLength: 600, BeamWidth: 18, Duration: 1200 * time.Millisecond, DamageInterval: 250 * time.Millisecond, Burst: 1, Directions: 1, MaxLevel: 8, PlayerAvailable: true, Levels: map[int][]SpellModifier{2: {{Attribute: "beam_length", Value: 100}}, 3: {{Attribute: "damage", Value: 3}}, 4: {{Attribute: "linger_duration_ms", Value: 400}}, 5: {{Attribute: "beam_width", Value: 8}}, 6: {{Attribute: "damage_interval_ms", Value: -50}}, 7: {{Attribute: "cooldown_ms", Value: -300}}, 8: {{Attribute: "directions", Value: 1}}}},
}

var characters = map[string]CharacterDefinition{
	"ranger":   {ID: "ranger", Name: "Ranger", SpriteID: "character-ranger", MaxHP: 100, MovementSpeed: 220, PickupRadius: 120, ResurrectionDuration: 2 * time.Second, ResurrectionRadius: 120, ResurrectionImmunityDuration: 5 * time.Second, DefaultSpellID: "fireball", StartingSpellIDs: []string{"fireball"}},
	"frieren":  {ID: "frieren", Name: "Frieren", SpriteID: "character-frieren", MaxHP: 90, MovementSpeed: 210, PickupRadius: 125, ResurrectionDuration: 2 * time.Second, ResurrectionRadius: 120, ResurrectionImmunityDuration: 5 * time.Second, DefaultSpellID: "soul-track", StartingSpellIDs: []string{"soul-track"}},
	"catapult": {ID: "catapult", Name: "Catapult", SpriteID: "character-catapult", MaxHP: 115, MovementSpeed: 195, PickupRadius: 115, ResurrectionDuration: 2 * time.Second, ResurrectionRadius: 120, ResurrectionImmunityDuration: 5 * time.Second, DefaultSpellID: "rocket", StartingSpellIDs: []string{"rocket"}},
}

var enemies = map[string]EnemyDefinition{
	"slime-stage-1": {ID: "slime-stage-1", Name: "Slime", SpriteID: "enemy-slime-stage-1", Score: 100, Experience: 1, MaxHP: 40, Armor: 1, Speed: 80, Radius: 24, ContactDamage: 10, SpellIDs: []string{"slime-punch"}, ContactDelay: 800 * time.Millisecond},
	"slime-stage-2": {ID: "slime-stage-2", Name: "Greater Slime", SpriteID: "enemy-slime-stage-2", Score: 250, Experience: 2, MaxHP: 90, Armor: 1, Speed: 92, Radius: 30, ContactDamage: 16, SpellIDs: []string{"enemy-slime-ball"}, ContactDelay: 750 * time.Millisecond},
	"slime-stage-3": {ID: "slime-stage-3", Name: "Slime King", SpriteID: "enemy-slime-stage-3", Score: 5000, Experience: 30, MaxHP: 1800, Armor: 1, Speed: 62, Radius: 54, ContactDamage: 28, SpellIDs: []string{"slime-punch"}, ContactDelay: 650 * time.Millisecond},
}

var levelOne = LevelDefinition{
	ID: "level-1", Name: "Slime Meadow", Duration: 15 * time.Minute,
	TerrainAssetIDs:  []string{"terrain-variant-1", "terrain-variant-2", "terrain-variant-3"},
	ObstacleAssetIDs: []string{"obstacle-large-rock-1", "obstacle-large-rock-2", "obstacle-large-rock-3"},
	Obstacles:        Obstacles,
	Events: []LevelEvent{
		{ID: "opening-slimes", At: 0, Type: "spawn_rate", Show: true, Title: "Slimes emerge", Description: "Small Slimes begin surrounding the squad.", SpawnRate: &SpawnRateDefinition{RatePerSecond: 1, MaxLiving: 60, Entries: []SpawnEntry{{EnemyID: "slime-stage-1", Weight: 100}}}},
		{ID: "greater-slimes", At: time.Minute, Type: "spawn_rate", Show: true, Title: "Greater Slimes", Description: "Normal spawns switch to faster, tougher Greater Slimes.", SpawnRate: &SpawnRateDefinition{RatePerSecond: 1.8, MaxLiving: 110, Entries: []SpawnEntry{{EnemyID: "slime-stage-2", Weight: 100}}}},
		{ID: "slime-surge", At: 3 * time.Minute, Type: "monster_buff", Show: true, Title: "Slime Surge", Description: "All Slimes gain 50% health and 20% movement speed.", MonsterBuff: &MonsterBuffDefinition{HealthMultiplier: 1.5, SpeedMultiplier: 1.2}},
		{ID: "slime-swarm", At: 4 * time.Minute, Type: "spawn_rate", Show: true, Title: "Slime Swarm", Description: "Greater Slimes emerge more rapidly.", SpawnRate: &SpawnRateDefinition{RatePerSecond: 2.4, MaxLiving: 150, Entries: []SpawnEntry{{EnemyID: "slime-stage-2", Weight: 100}}}},
		{ID: "slime-king", At: 14 * time.Minute, Type: "boss", Show: true, Title: "Slime King", Description: "Defeat the empowered Slime King to end the run early.", EnemyID: "slime-stage-3", EndMatchOnDeath: true, BossMultipliers: &EnemyStatMultipliers{Health: 5, MovementSpeed: 1.2, AttackDamage: 2}},
		{ID: "level-complete", At: 15 * time.Minute, Type: "end", Show: true, Title: "Dawn", Description: "The run ends and the final score is calculated."},
	},
}

var levelDefinitions = map[string]LevelDefinition{levelOne.ID: levelOne}

func LevelByID(id string) (LevelDefinition, bool) {
	if id == "" {
		id = "level-1"
	}
	value, ok := levelDefinitions[id]
	return value, ok
}
func AvailableLevels() []LevelDefinition {
	ids := make([]string, 0, len(levelDefinitions))
	for id := range levelDefinitions {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	result := make([]LevelDefinition, 0, len(ids))
	for _, id := range ids {
		result = append(result, levelDefinitions[id])
	}
	return result
}
func CharacterByID(id string) (CharacterDefinition, bool) {
	value, ok := characters[id]
	return value, ok
}
func AvailableCharacters() []CharacterDefinition {
	ids := make([]string, 0, len(characters))
	for id := range characters {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	result := make([]CharacterDefinition, 0, len(ids))
	for _, id := range ids {
		result = append(result, characters[id])
	}
	return result
}
func SpellByID(id string) (SpellDefinition, bool) { value, ok := spells[id]; return value, ok }
func EnemyByID(id string) (EnemyDefinition, bool) { value, ok := enemies[id]; return value, ok }
