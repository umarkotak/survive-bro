package simulation

import (
	"math"
	"testing"
	"time"

	"survive-bro/apps/backend/internal/protocol"
)

func TestInputIsNormalizedAndMovementIsAuthoritative(t *testing.T) {
	now := time.Unix(100, 0)
	match := NewMatch(now, 1)
	player := match.AddPlayer("p1", "Umar", now)
	startX, startY := player.X, player.Y

	if err := match.ApplyInput("p1", protocol.InputPayload{Sequence: 1, MoveX: 1, MoveY: 1}, now); err != nil {
		t.Fatalf("ApplyInput() error = %v", err)
	}
	match.Step(now.Add(TickDuration))

	distance := math.Hypot(player.X-startX, player.Y-startY)
	if math.Abs(distance-PlayerSpeed*TickDuration.Seconds()) > 0.001 {
		t.Fatalf("distance = %f, want normalized movement %f", distance, PlayerSpeed*TickDuration.Seconds())
	}
	if player.LastProcessedInput != 1 || player.Facing != "right" {
		t.Fatalf("unexpected input state: %#v", player)
	}
}

func TestStaleInputStopsMovement(t *testing.T) {
	now := time.Unix(100, 0)
	match := NewMatch(now, 1)
	player := match.AddPlayer("p1", "Umar", now)
	if err := match.ApplyInput("p1", protocol.InputPayload{Sequence: 1, MoveX: 1}, now); err != nil {
		t.Fatalf("ApplyInput() error = %v", err)
	}
	match.Step(now.Add(InputStaleAfter + time.Millisecond))
	if player.VelocityX != 0 || player.VelocityY != 0 {
		t.Fatalf("stale velocity = (%f,%f)", player.VelocityX, player.VelocityY)
	}
}

func TestRejectsInvalidAxes(t *testing.T) {
	now := time.Unix(100, 0)
	match := NewMatch(now, 1)
	match.AddPlayer("p1", "Umar", now)
	for _, input := range []protocol.InputPayload{
		{Sequence: 1, MoveX: 1.1},
		{Sequence: 1, MoveX: math.NaN()},
		{Sequence: 1, MoveY: math.Inf(1)},
	} {
		if err := match.ApplyInput("p1", input, now); err == nil {
			t.Fatalf("ApplyInput(%#v) error = nil", input)
		}
	}
}

func TestSnapshotContainsAllPlayersAndSharedTeamState(t *testing.T) {
	now := time.Unix(100, 0)
	match := NewMatch(now, 1)
	match.AddPlayer("p2", "Budi", now)
	match.AddPlayer("p1", "Umar", now)
	snapshot := match.Snapshot(now)
	if len(snapshot.Players) != 2 || snapshot.Players[0].ID != "p1" || snapshot.Players[1].ID != "p2" {
		t.Fatalf("unexpected players: %#v", snapshot.Players)
	}
	if snapshot.Team.Level != 1 || snapshot.Team.ExperienceRequired != 13 || snapshot.RemainingMs != MatchDuration.Milliseconds() {
		t.Fatalf("unexpected team snapshot: %#v", snapshot)
	}
}

func TestPlayerCannotMoveThroughRock(t *testing.T) {
	now := time.Unix(100, 0)
	match := NewMatch(now, 1)
	player := match.AddPlayer("p1", "Umar", now)
	player.X = Obstacles[0].X
	player.Y = Obstacles[0].Y
	match.Step(now.Add(TickDuration))
	distance := math.Hypot(player.X-Obstacles[0].X, player.Y-Obstacles[0].Y)
	if distance < PlayerRadius+Obstacles[0].Radius {
		t.Fatalf("player remained inside obstacle: distance=%f", distance)
	}
}

func TestSoloPlayerAutomaticallyResurrectsAtHalfHealthWithImmunity(t *testing.T) {
	now := time.Unix(100, 0)
	match := NewMatch(now, 1)
	player := match.AddPlayer("p1", "Umar", now)
	if match.TeamLives != 1 {
		t.Fatalf("team lives = %d, want 1", match.TeamLives)
	}

	match.applyPlayerDamage(player, player.MaxHP)
	if player.Alive || !player.ResurrectionPending || match.TeamLives != 1 {
		t.Fatalf("unexpected death state: player=%#v lives=%d", player, match.TeamLives)
	}
	for range int(player.ResurrectionDuration / TickDuration) {
		match.updateResurrections()
	}
	if !player.Alive || player.HP != 50 || player.ResurrectionPending {
		t.Fatalf("unexpected resurrection state: %#v", player)
	}
	if match.TeamLives != 0 {
		t.Fatalf("team lives after resurrection = %d, want 0", match.TeamLives)
	}

	match.applyPlayerDamage(player, 10)
	if player.HP != 50 {
		t.Fatalf("immune hp = %d, want 50", player.HP)
	}
	match.Elapsed = player.ImmuneUntil
	match.applyPlayerDamage(player, 10)
	if player.HP != 40 {
		t.Fatalf("post-immunity hp = %d, want 40", player.HP)
	}
}

func TestMultiplayerResurrectionRequiresNearbyLivingTeammate(t *testing.T) {
	now := time.Unix(100, 0)
	match := NewMatch(now, 1)
	dead := match.AddPlayer("p1", "Umar", now)
	friend := match.AddPlayer("p2", "Budi", now)
	friend.X, friend.Y = dead.X+dead.ResurrectionRadius+1, dead.Y
	match.applyPlayerDamage(dead, dead.MaxHP)

	match.updateResurrections()
	if dead.ResurrectionProgress != 0 {
		t.Fatalf("progress without nearby friend = %s", dead.ResurrectionProgress)
	}
	friend.X = dead.X + dead.ResurrectionRadius - 1
	match.updateResurrections()
	if dead.ResurrectionProgress != TickDuration {
		t.Fatalf("nearby progress = %s, want %s", dead.ResurrectionProgress, TickDuration)
	}
	friend.X = dead.X + dead.ResurrectionRadius + 1
	match.updateResurrections()
	if dead.ResurrectionProgress != 0 {
		t.Fatalf("progress after leaving radius = %s, want 0", dead.ResurrectionProgress)
	}
	friend.X = dead.X
	for range int(dead.ResurrectionDuration / TickDuration) {
		match.updateResurrections()
	}
	if !dead.Alive || dead.HP != 50 {
		t.Fatalf("unexpected resurrected player: %#v", dead)
	}
}

func TestMultiplayerMatchEndsWhenNoLivingReviverRemains(t *testing.T) {
	now := time.Unix(100, 0)
	match := NewMatch(now, 1)
	first := match.AddPlayer("p1", "Umar", now)
	second := match.AddPlayer("p2", "Budi", now)
	match.applyPlayerDamage(first, first.MaxHP)
	match.applyPlayerDamage(second, second.MaxHP)

	events := match.Step(now.Add(TickDuration))
	if !match.Finished || events.MatchEnded == nil || events.MatchEnded.Outcome != "lost" {
		t.Fatalf("match did not end after squad wipe: finished=%v event=%#v", match.Finished, events.MatchEnded)
	}
}
