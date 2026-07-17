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

func TestMonsterMeleeSpellDamagesOnlyWithinRangeAndRespectsCooldown(t *testing.T) {
	now := time.Unix(100, 0)
	match := NewMatch(now, 1)
	player := match.AddPlayer("p1", "Umar", now)
	player.ArmorPercent = 0.5
	monster := &Monster{ID: 1, X: player.X + 90, Y: player.Y, SpellIDs: []string{"slime-punch"}, LastSpellAt: make(map[string]time.Duration), AttackDamageMultiplier: 1}
	match.Elapsed = 1200 * time.Millisecond

	events := Events{}
	match.castMonsterSpell(monster, player, &events)
	if player.HP != player.MaxHP-7 {
		t.Fatalf("melee hp = %d, want %d", player.HP, player.MaxHP-7)
	}
	if len(match.Projectiles) != 0 {
		t.Fatalf("melee created %d projectiles", len(match.Projectiles))
	}
	if len(events.MonsterAttacks) != 1 || events.MonsterAttacks[0].SpellID != "slime-punch" || events.MonsterAttacks[0].TargetPlayerID != player.ID {
		t.Fatalf("melee attack events = %#v", events.MonsterAttacks)
	}

	match.castMonsterSpell(monster, player, &Events{})
	if player.HP != player.MaxHP-7 {
		t.Fatalf("melee ignored cooldown: hp = %d", player.HP)
	}

	match.Elapsed += 1200 * time.Millisecond
	monster.X = player.X + 91
	match.castMonsterSpell(monster, player, &Events{})
	if player.HP != player.MaxHP-7 {
		t.Fatalf("out-of-range melee changed hp to %d", player.HP)
	}
}

func TestSpellChestChoiceLearnsUniqueSpellAndRejectsDuplicate(t *testing.T) {
	now := time.Unix(100, 0)
	match := NewMatch(now, 1)
	player := match.AddPlayer("p1", "Umar", now)
	_, ok := SpellByID("heavy-aura")
	if !ok {
		t.Skip("runtime spell data is not loaded")
	}

	first := match.applySpellChoice(player, "heavy-aura")
	if first.FinalValue != 1 || player.SpellID != "fireball" || player.SpellLevels["heavy-aura"] != 1 {
		t.Fatalf("unexpected learned spell: event=%#v player=%#v", first, player)
	}
	second := match.applySpellChoice(player, "heavy-aura")
	if second.AddedValue != 0 || player.SpellLevels["heavy-aura"] != 1 || len(player.SpellIDs) != 2 {
		t.Fatalf("duplicate spell was added: event=%#v spells=%#v", second, player.SpellIDs)
	}
}

func TestSpellChestExcludesOwnedSpellsAndFallsBackToTreasure(t *testing.T) {
	now := time.Unix(100, 0)
	match := NewMatch(now, 1)
	player := match.AddPlayer("p1", "Umar", now)
	match.applySpellChoice(player, "heavy-aura")
	pool := []string{"fireball", "heavy-aura", "meteorite", "tracking-beam"}
	offer := match.createPlayerOffer(player, "spell_chest", pool)
	for _, choice := range offer.Choices {
		if choice.Attribute == "spell:heavy-aura" {
			t.Fatalf("owned spell appeared in offer: %#v", offer)
		}
	}
	match.applySpellChoice(player, "meteorite")
	match.applySpellChoice(player, "tracking-beam")
	offer = match.createPlayerOffer(player, "spell_chest", pool)
	if offer.Source != "treasure_chest" || len(offer.Choices) != 3 {
		t.Fatalf("full spell inventory did not fall back: %#v", offer)
	}
}

func TestOwnedSpellsCastOnIndependentCooldowns(t *testing.T) {
	now := time.Unix(100, 0)
	match := NewMatch(now, 1)
	player := match.AddPlayer("p1", "Umar", now)
	match.applySpellChoice(player, "heavy-aura")
	monster := &Monster{ID: 1, X: player.X + 50, Y: player.Y, HP: 1000, MaxHP: 1000, Armor: 1}
	match.Monsters[monster.ID] = monster
	match.Elapsed = 3 * time.Second
	events := Events{}
	match.updateWeapons(&events)
	if len(match.Projectiles) == 0 || len(match.Explosions) == 0 {
		t.Fatalf("owned spells did not both cast: projectiles=%d auras=%d", len(match.Projectiles), len(match.Explosions))
	}
}

func TestHeavyAuraFollowsOwnerAndDamagesOnInterval(t *testing.T) {
	now := time.Unix(100, 0)
	match := NewMatch(now, 1)
	player := match.AddPlayer("p1", "Umar", now)
	spell, ok := SpellByID("heavy-aura")
	if !ok {
		t.Skip("runtime spell data is not loaded")
	}
	match.applySpellChoice(player, "heavy-aura")
	monster := &Monster{ID: 1, X: player.X + 50, Y: player.Y, HP: 100, MaxHP: 100, Armor: 1}
	match.Monsters[monster.ID] = monster
	match.Elapsed = spell.Cooldown
	events := Events{}
	match.updateWeapons(&events)
	match.updateExplosions(&events, &events.Metrics)
	if monster.HP != 100-(spell.Damage-1) {
		t.Fatalf("aura hp = %d", monster.HP)
	}
	player.X += 20
	match.Elapsed += TickDuration
	match.updateExplosions(&events, &events.Metrics)
	for _, aura := range match.Explosions {
		if aura.X != player.X {
			t.Fatalf("aura x = %f, player x = %f", aura.X, player.X)
		}
	}
}

func TestHeavyAuraExistsWithoutTargets(t *testing.T) {
	now := time.Unix(100, 0)
	match := NewMatch(now, 1)
	player := match.AddPlayer("p1", "Umar", now)
	match.applySpellChoice(player, "heavy-aura")
	match.updateWeapons(&Events{})
	if len(match.Explosions) != 1 {
		t.Fatalf("always-on aura count = %d, want 1", len(match.Explosions))
	}
}

func TestSpellLevelUpgradeTargetsNamedSpell(t *testing.T) {
	now := time.Unix(100, 0)
	match := NewMatch(now, 1)
	player := match.AddPlayer("p1", "Umar", now)
	match.applySpellChoice(player, "heavy-aura")
	event := match.applyUpgrade(player, "level_up", "spell:heavy-aura:level")
	if event.Attribute != "spell:heavy-aura:level" || player.SpellLevels["heavy-aura"] != 2 || player.SpellLevels["fireball"] != 1 {
		t.Fatalf("targeted spell upgrade event=%#v levels=%#v", event, player.SpellLevels)
	}
}

func TestMeteoriteDamagesMonstersButNotPlayers(t *testing.T) {
	now := time.Unix(100, 0)
	match := NewMatch(now, 1)
	player := match.AddPlayer("p1", "Umar", now)
	spell, ok := SpellByID("meteorite")
	if !ok {
		t.Skip("runtime spell data is not loaded")
	}
	match.applySpellChoice(player, "meteorite")
	monster := &Monster{ID: 1, X: player.X + 100, Y: player.Y, HP: 100, MaxHP: 100, Armor: 1}
	match.Monsters[monster.ID] = monster
	match.Elapsed = spell.Cooldown
	events := Events{}
	match.updateWeapons(&events)
	match.Elapsed += spell.Duration
	match.updateMeteors(&events)
	if monster.HP != 100-(spell.Damage-1) || player.HP != player.MaxHP {
		t.Fatalf("meteor result: monster hp=%d player hp=%d", monster.HP, player.HP)
	}
}

func TestTrackingBeamRetargetsAfterTargetDies(t *testing.T) {
	now := time.Unix(100, 0)
	match := NewMatch(now, 1)
	player := match.AddPlayer("p1", "Umar", now)
	spell, ok := SpellByID("tracking-beam")
	if !ok {
		t.Skip("runtime spell data is not loaded")
	}
	match.applySpellChoice(player, "tracking-beam")
	first := &Monster{ID: 1, X: player.X + 100, Y: player.Y, HP: 100, MaxHP: 100, Armor: 1}
	second := &Monster{ID: 2, X: player.X + 200, Y: player.Y, HP: 100, MaxHP: 100, Armor: 1}
	match.Monsters[first.ID], match.Monsters[second.ID] = first, second
	match.Elapsed = spell.Cooldown
	events := Events{}
	match.updateWeapons(&events)
	delete(match.Monsters, first.ID)
	match.updateBeams(&events, &events.Metrics)
	for _, beam := range match.Beams {
		if beam.TargetID != second.ID {
			t.Fatalf("tracking beam target = %d, want %d", beam.TargetID, second.ID)
		}
	}
}

func TestTreasureRateEventChangesKillCadence(t *testing.T) {
	now := time.Unix(100, 0)
	level := LevelDefinition{ID: "treasure-test", Duration: time.Minute, Events: []LevelEvent{{ID: "faster-treasure", Type: "treasure_rate", TreasureRate: &TreasureRateDefinition{KillsPerChest: 2}}}}
	match := NewMatch(now, 1, level)
	match.AddPlayer("p1", "Umar", now)
	for id := uint64(1); id <= 2; id++ {
		match.Monsters[id] = &Monster{ID: id, X: 100, Y: 100, Experience: 1, Score: 1}
		match.killMonster(id, "p1", &Events{})
	}
	crates := 0
	for _, pickup := range match.Pickups {
		if pickup.Kind == "power_crate" {
			crates++
		}
	}
	if match.TreasureEveryKills != 2 || crates != 1 {
		t.Fatalf("treasure cadence=%d crates=%d", match.TreasureEveryKills, crates)
	}
}
