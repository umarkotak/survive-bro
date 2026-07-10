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
