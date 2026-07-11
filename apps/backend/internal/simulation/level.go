package simulation

import "time"

type SpellDefinition struct {
	ID              string
	Kind            string
	Damage          int
	Cooldown        time.Duration
	ProjectileSpeed float64
	Range           float64
	Radius          float64
	Burst           int
	Directions      int
	BeamLength      float64
	BeamWidth       float64
	Duration        time.Duration
	DamageInterval  time.Duration
}

type CharacterDefinition struct {
	ID                 string
	Name               string
	SpriteID           string
	MaxHP              int
	ArmorPercent       float64
	MovementSpeed      float64
	HealthRegeneration float64
	AttackBuffPercent  float64
	CooldownPercent    float64
	PickupRadius       float64
	BaseSpellID        string
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

type LevelEvent struct {
	ID          string
	At          time.Duration
	Type        string
	Title       string
	Description string
	SpawnRate   *SpawnRateDefinition
	MonsterBuff *MonsterBuffDefinition
	EnemyID     string
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
	"fireball":   {ID: "fireball", Kind: "projectile", Damage: 20, Cooldown: 750 * time.Millisecond, ProjectileSpeed: 700, Range: 700, Radius: 10, Burst: 1, Directions: 1},
	"soul-track": {ID: "soul-track", Kind: "beam", Damage: 18, Cooldown: 1500 * time.Millisecond, Range: 520, Radius: 16, Burst: 1, Directions: 1, BeamLength: 520, BeamWidth: 32, Duration: time.Second, DamageInterval: 500 * time.Millisecond},
}

var characters = map[string]CharacterDefinition{
	"ranger":  {ID: "ranger", Name: "Ranger", SpriteID: "character-ranger", MaxHP: 100, MovementSpeed: 220, PickupRadius: 120, BaseSpellID: "fireball"},
	"frieren": {ID: "frieren", Name: "Frieren", SpriteID: "character-frieren", MaxHP: 90, MovementSpeed: 210, PickupRadius: 125, BaseSpellID: "soul-track"},
}

var enemies = map[string]EnemyDefinition{
	"slime-stage-1": {ID: "slime-stage-1", Name: "Slime", SpriteID: "enemy-slime-stage-1", Score: 100, Experience: 1, MaxHP: 40, Speed: 80, Radius: 24, ContactDamage: 10, ContactDelay: 800 * time.Millisecond},
	"slime-stage-2": {ID: "slime-stage-2", Name: "Greater Slime", SpriteID: "enemy-slime-stage-2", Score: 250, Experience: 2, MaxHP: 90, Speed: 92, Radius: 30, ContactDamage: 16, ContactDelay: 750 * time.Millisecond},
	"slime-stage-3": {ID: "slime-stage-3", Name: "Slime King", SpriteID: "enemy-slime-stage-3", Score: 5000, Experience: 30, MaxHP: 1800, Speed: 62, Radius: 54, ContactDamage: 28, ContactDelay: 650 * time.Millisecond},
}

var levelOne = LevelDefinition{
	ID: "level-1", Name: "Slime Meadow", Duration: 6 * time.Minute,
	TerrainAssetIDs:  []string{"terrain-variant-1", "terrain-variant-2", "terrain-variant-3"},
	ObstacleAssetIDs: []string{"obstacle-large-rock-1", "obstacle-large-rock-2", "obstacle-large-rock-3"},
	Obstacles:        Obstacles,
	Events: []LevelEvent{
		{ID: "opening-slimes", At: 0, Type: "spawn_rate", Title: "Slimes emerge", Description: "Small Slimes begin surrounding the squad.", SpawnRate: &SpawnRateDefinition{RatePerSecond: 1, MaxLiving: 60, Entries: []SpawnEntry{{EnemyID: "slime-stage-1", Weight: 100}}}},
		{ID: "greater-slimes", At: time.Minute, Type: "spawn_rate", Title: "Greater Slimes", Description: "Normal spawns switch to faster, tougher Greater Slimes.", SpawnRate: &SpawnRateDefinition{RatePerSecond: 1.8, MaxLiving: 110, Entries: []SpawnEntry{{EnemyID: "slime-stage-2", Weight: 100}}}},
		{ID: "slime-surge", At: 3 * time.Minute, Type: "monster_buff", Title: "Slime Surge", Description: "All Slimes gain 50% health and 20% movement speed.", MonsterBuff: &MonsterBuffDefinition{HealthMultiplier: 1.5, SpeedMultiplier: 1.2}},
		{ID: "slime-swarm", At: 4 * time.Minute, Type: "spawn_rate", Title: "Slime Swarm", Description: "Greater Slimes emerge more rapidly.", SpawnRate: &SpawnRateDefinition{RatePerSecond: 2.4, MaxLiving: 150, Entries: []SpawnEntry{{EnemyID: "slime-stage-2", Weight: 100}}}},
		{ID: "slime-king", At: 5 * time.Minute, Type: "boss", Title: "Slime King", Description: "The Slime King enters the meadow.", EnemyID: "slime-stage-3"},
		{ID: "level-complete", At: 6 * time.Minute, Type: "end", Title: "Dawn", Description: "The run ends and the final score is calculated."},
	},
}

func LevelByID(id string) (LevelDefinition, bool) {
	if id == "" || id == levelOne.ID {
		return levelOne, true
	}
	return LevelDefinition{}, false
}
func AvailableLevels() []LevelDefinition { return []LevelDefinition{levelOne} }
func CharacterByID(id string) (CharacterDefinition, bool) {
	value, ok := characters[id]
	return value, ok
}
func AvailableCharacters() []CharacterDefinition {
	return []CharacterDefinition{characters["ranger"], characters["frieren"]}
}
func SpellByID(id string) (SpellDefinition, bool) { value, ok := spells[id]; return value, ok }
func EnemyByID(id string) (EnemyDefinition, bool) { value, ok := enemies[id]; return value, ok }
