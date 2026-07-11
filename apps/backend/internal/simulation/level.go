package simulation

import "time"

type EnemyDefinition struct {
	ID            string
	MaxHP         int
	Speed         float64
	Radius        float64
	ContactDamage int
	ContactDelay  time.Duration
	Experience    int
}

type LevelEvent struct {
	At      time.Duration
	Type    string
	EnemyID string
}

type LevelDefinition struct {
	ID               string
	Name             string
	Duration         time.Duration
	TerrainAssetIDs  []string
	ObstacleAssetIDs []string
	Obstacles        []Obstacle
	Enemies          map[string]EnemyDefinition
	InitialEnemyID   string
	Events           []LevelEvent
}

var levelOne = LevelDefinition{
	ID: "level-1", Name: "Slime Meadow", Duration: 6 * time.Minute,
	TerrainAssetIDs:  []string{"terrain-variant-1", "terrain-variant-2", "terrain-variant-3"},
	ObstacleAssetIDs: []string{"obstacle-large-rock-1", "obstacle-large-rock-2", "obstacle-large-rock-3"},
	Obstacles:        Obstacles,
	Enemies: map[string]EnemyDefinition{
		"slime-stage-1": {ID: "slime-stage-1", MaxHP: 40, Speed: 80, Radius: 24, ContactDamage: 10, ContactDelay: 800 * time.Millisecond, Experience: 1},
		"slime-stage-2": {ID: "slime-stage-2", MaxHP: 90, Speed: 92, Radius: 30, ContactDamage: 16, ContactDelay: 750 * time.Millisecond, Experience: 2},
		"slime-stage-3": {ID: "slime-stage-3", MaxHP: 1800, Speed: 62, Radius: 54, ContactDamage: 28, ContactDelay: 650 * time.Millisecond, Experience: 30},
	},
	InitialEnemyID: "slime-stage-1",
	Events: []LevelEvent{
		{At: time.Minute, Type: "enemy_stage", EnemyID: "slime-stage-2"},
		{At: 5 * time.Minute, Type: "boss", EnemyID: "slime-stage-3"},
		{At: 6 * time.Minute, Type: "end"},
	},
}

func LevelByID(id string) (LevelDefinition, bool) {
	if id == "" || id == levelOne.ID {
		return levelOne, true
	}
	return LevelDefinition{}, false
}

func AvailableLevels() []LevelDefinition { return []LevelDefinition{levelOne} }
