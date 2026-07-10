package simulation

import "time"

const (
	TickDuration         = 50 * time.Millisecond
	SnapshotEveryTicks   = 2
	MatchDuration        = 5 * time.Minute
	InputStaleAfter      = 250 * time.Millisecond
	WorldWidth           = 3200.0
	WorldHeight          = 1800.0
	PlayerSpawnX         = 1600.0
	PlayerSpawnY         = 900.0
	PlayerSpawnRadius    = 80.0
	PlayerRadius         = 30.0
	PlayerMaxHP          = 100
	PlayerSpeed          = 220.0
	PlayerPickupRadius   = 120.0
	LevelMovementBonus   = 0.08
	LevelArmorBonus      = 0.10
	LevelMagnetBonus     = 0.15
	MaximumMovementBonus = 0.80
	MaximumArmorPercent  = 0.60
	MaximumPickupRadius  = 600.0
	WeaponDamage         = 20
	WeaponCooldown       = 750 * time.Millisecond
	ProjectileSpeed      = 700.0
	ProjectileSpeedLevel = 70.0
	ProjectileSpread     = 10.0
	MaximumProjectiles   = 4
	ProjectileRange      = 700.0
	ProjectileRadius     = 10.0
	PickupAttractSpeed   = 900.0
	PickupCollectRadius  = 32.0
	PowerCrateEveryKills = 12
	PowerCrateRadius     = 70.0
	PowerMaximumStacks   = 5
	PowerHasteBonus      = 0.08
	PowerArmorBonus      = 0.10
	PowerMagnetBonus     = 60.0
	MonsterMaxHP         = 40
	MonsterSpeed         = 80.0
	MonsterRadius        = 24.0
	MonsterContactDamage = 10
	MonsterContactDelay  = 800 * time.Millisecond
	MonsterXP            = 1
)

func ProjectileSpeedAtLevel(level int) float64 {
	return ProjectileSpeed + float64(max(0, level-1))*ProjectileSpeedLevel
}

func ProjectileCountAtLevel(level int) int {
	return min(MaximumProjectiles, max(1, level))
}

type Obstacle struct {
	ID     string
	Type   string
	X      float64
	Y      float64
	Radius float64
}

var Obstacles = []Obstacle{
	{ID: "rock-1", Type: "large_rock", X: 480, Y: 360, Radius: 65},
	{ID: "rock-2", Type: "large_rock", X: 930, Y: 280, Radius: 65},
	{ID: "rock-3", Type: "large_rock", X: 1380, Y: 420, Radius: 65},
	{ID: "rock-4", Type: "large_rock", X: 2140, Y: 330, Radius: 65},
	{ID: "rock-5", Type: "large_rock", X: 2750, Y: 430, Radius: 65},
	{ID: "rock-6", Type: "large_rock", X: 580, Y: 1260, Radius: 65},
	{ID: "rock-7", Type: "large_rock", X: 1080, Y: 1480, Radius: 65},
	{ID: "rock-8", Type: "large_rock", X: 2030, Y: 1390, Radius: 65},
	{ID: "rock-9", Type: "large_rock", X: 2600, Y: 1250, Radius: 65},
	{ID: "rock-10", Type: "large_rock", X: 2920, Y: 900, Radius: 65},
}
