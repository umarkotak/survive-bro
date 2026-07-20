package simulation

import (
	"testing"
	"time"
)

func TestRuntimeLoadsMovementVariants(t *testing.T) {
	if err := LoadRuntimeGameData("../../../../game-data/game.json"); err != nil {
		t.Fatalf("LoadRuntimeGameData error: %v", err)
	}
	darter, ok := EnemyByID("slime-darter")
	if !ok {
		t.Fatal("slime-darter missing")
	}
	if darter.Movement.Mode != "dash" || darter.Movement.DashSpeedMultiplier <= 1 {
		t.Fatalf("darter movement = %#v", darter.Movement)
	}
	if _, ok := EnemyByID("slime-sprinter"); !ok {
		t.Fatal("slime-sprinter missing")
	}
	if _, ok := EnemyByID("slime-bruiser"); !ok {
		t.Fatal("slime-bruiser missing")
	}
	if stage1, _ := EnemyByID("slime-stage-1"); stage1.Movement.Mode != "normal" {
		t.Fatalf("stage1 mode = %q", stage1.Movement.Mode)
	}
}

func TestDasherMonsterCompletesDashCycle(t *testing.T) {
	if err := LoadRuntimeGameData("../../../../game-data/game.json"); err != nil {
		t.Fatalf("LoadRuntimeGameData error: %v", err)
	}
	def, _ := EnemyByID("slime-darter")
	now := time.Unix(100, 0)
	match := NewMatch(now, 42)
	player := match.AddPlayer("p1", "Umar", now)
	player.X, player.Y = 1500, 900
	monster := &Monster{
		ID: 1, TypeID: "slime-darter", X: player.X - 400, Y: player.Y,
		HP: 1000, MaxHP: 1000, Speed: def.Speed, Radius: def.Radius,
		MovementMode:        def.Movement.Mode,
		DashInterval:        def.Movement.DashInterval,
		DashWindup:          def.Movement.DashWindup,
		DashDuration:        def.Movement.DashDuration,
		DashSpeedMultiplier: def.Movement.DashSpeedMultiplier,
		LastContact:         map[string]time.Duration{},
		LastSpellAt:         map[string]time.Duration{},
	}
	match.Monsters[1] = monster

	advance := func(ticks int) {
		for range ticks {
			match.Elapsed += TickDuration
			match.moveMonster(monster, player)
		}
	}

	advance(int(def.Movement.DashInterval/TickDuration) + 1)
	if monster.DashState != "windup" {
		t.Fatalf("after idle phase dashState = %q, want windup", monster.DashState)
	}

	advance(int(def.Movement.DashWindup/TickDuration) + 1)
	if monster.DashState != "dash" {
		t.Fatalf("after windup dashState = %q, want dash", monster.DashState)
	}
	if monster.DashDirX <= 0 {
		t.Fatalf("dash direction not locked toward player: dirX=%v", monster.DashDirX)
	}

	startX := monster.X
	advance(int(def.Movement.DashDuration / TickDuration))
	if monster.DashState != "idle" {
		t.Fatalf("after dash dashState = %q, want idle", monster.DashState)
	}
	normalStep := def.Speed * TickDuration.Seconds()
	dashStep := def.Speed * def.Movement.DashSpeedMultiplier * TickDuration.Seconds()
	if dashStep <= normalStep {
		t.Fatalf("dash step %v not faster than normal %v", dashStep, normalStep)
	}
	if monster.X-startX < dashStep*0.5 {
		t.Fatalf("dash lunge distance %v too small, expected ~%v burst", monster.X-startX, dashStep)
	}
}
