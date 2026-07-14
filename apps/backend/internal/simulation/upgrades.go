package simulation

import (
	"time"

	"survive-bro/apps/backend/internal/protocol"
)

func (m *Match) beginUpgradePhase(now time.Time, source string) {
	m.nextUpgradeOfferID++
	phase := &UpgradePhase{
		ID:       m.nextUpgradeOfferID,
		Source:   source,
		Deadline: now.Add(UpgradeSelectionTimeout),
		Offers:   make(map[string]*PlayerUpgradeOffer, len(m.Players)),
	}
	for _, playerID := range sortedPlayerIDs(m.Players) {
		player := m.Players[playerID]
		phase.Offers[playerID] = &PlayerUpgradeOffer{Choices: m.createUpgradeChoices(player, source), SelectedIndex: -1}
		player.MoveX, player.MoveY = 0, 0
	}
	m.UpgradePhase = phase
}

func (m *Match) beginEarnedLevelIfReady(now time.Time) bool {
	required := RequiredExperience(m.TeamLevel)
	if m.UpgradePhase != nil || m.TeamExperience < required {
		return false
	}
	m.TeamExperience -= required
	m.TeamLevel++
	m.beginUpgradePhase(now, "level_up")
	return true
}

func (m *Match) createUpgradeChoices(player *Player, source string) []protocol.UpgradeChoice {
	available := append([]string(nil), availableUpgradeAttributes(player)...)
	m.rng.Shuffle(len(available), func(i, j int) { available[i], available[j] = available[j], available[i] })
	choices := make([]protocol.UpgradeChoice, 0, 3)
	for _, attribute := range available[:min(3, len(available))] {
		copy := *player
		applied := m.applyUpgrade(&copy, source, attribute)
		choices = append(choices, protocol.UpgradeChoice{
			Attribute:    attribute,
			CurrentValue: currentUpgradeValue(player, attribute),
			AddedValue:   applied.AddedValue,
			FinalValue:   applied.FinalValue,
		})
	}
	return choices
}

func currentUpgradeValue(player *Player, attribute string) float64 {
	switch attribute {
	case "max_health":
		return float64(player.MaxHP)
	case "armor":
		return player.ArmorPercent
	case "movement_speed":
		return player.MovementSpeed
	case "health_regeneration":
		return player.HealthRegeneration
	case "attack_buff":
		return player.AttackBuffPercent
	case "cooldown":
		return player.CooldownPercent
	case "spell_damage":
		return float64(player.SpellDamage)
	case "projectile_speed":
		return player.ProjectileSpeed
	case "spell_burst":
		return float64(player.SpellBurst)
	case "spell_directions":
		return float64(player.SpellDirections)
	case "beam_length":
		return player.BeamLength
	case "beam_width":
		return player.BeamWidth
	case "spell_duration":
		return float64(player.SpellDuration.Milliseconds())
	case "explosion_radius":
		return player.ExplosionRadius
	case "explosion_duration":
		return float64(player.ExplosionDuration.Milliseconds())
	default:
		return 0
	}
}

func (m *Match) UpgradeOfferFor(playerID string) (protocol.UpgradeOfferedPayload, bool) {
	if m.UpgradePhase == nil {
		return protocol.UpgradeOfferedPayload{}, false
	}
	offer, ok := m.UpgradePhase.Offers[playerID]
	if !ok {
		return protocol.UpgradeOfferedPayload{}, false
	}
	pending := 0
	for _, candidate := range m.UpgradePhase.Offers {
		if candidate.SelectedIndex < 0 {
			pending++
		}
	}
	return protocol.UpgradeOfferedPayload{
		OfferID:      m.UpgradePhase.ID,
		Source:       m.UpgradePhase.Source,
		TeamLevel:    m.TeamLevel,
		DeadlineMs:   m.UpgradePhase.Deadline.UnixMilli(),
		PendingCount: pending,
		TotalCount:   len(m.UpgradePhase.Offers),
		Selected:     offer.SelectedIndex >= 0,
		Choices:      append([]protocol.UpgradeChoice(nil), offer.Choices...),
	}, true
}

func (m *Match) SelectUpgrade(playerID string, offerID uint64, choiceIndex int) (Events, error) {
	if m.UpgradePhase == nil {
		return Events{}, ErrNoUpgradePhase
	}
	if m.UpgradePhase.ID != offerID {
		return Events{}, ErrStaleUpgradeOffer
	}
	offer, ok := m.UpgradePhase.Offers[playerID]
	if !ok || choiceIndex < 0 || choiceIndex >= len(offer.Choices) {
		return Events{}, ErrInvalidUpgrade
	}
	if offer.SelectedIndex >= 0 {
		return Events{}, ErrUpgradeSelected
	}
	offer.SelectedIndex = choiceIndex
	events := Events{UpgradeOffersChanged: true}
	if m.upgradePhaseComplete() {
		m.resolveUpgradePhase(&events)
	}
	return events, nil
}

func (m *Match) upgradePhaseComplete() bool {
	if m.UpgradePhase == nil || len(m.UpgradePhase.Offers) == 0 {
		return false
	}
	for _, offer := range m.UpgradePhase.Offers {
		if offer.SelectedIndex < 0 {
			return false
		}
	}
	return true
}

func (m *Match) resolveUpgradePhase(events *Events) {
	phase := m.UpgradePhase
	if phase == nil {
		return
	}
	for _, playerID := range sortedPlayerIDs(m.Players) {
		offer, ok := phase.Offers[playerID]
		if !ok || len(offer.Choices) == 0 {
			continue
		}
		choiceIndex := offer.SelectedIndex
		if choiceIndex < 0 || choiceIndex >= len(offer.Choices) {
			choiceIndex = 0
		}
		events.AppliedUpgrades = append(events.AppliedUpgrades, m.applyUpgrade(m.Players[playerID], phase.Source, offer.Choices[choiceIndex].Attribute))
	}
	m.UpgradePhase = nil
	events.UpgradeOffersChanged = true
}

func (m *Match) DebugLevelUp(now time.Time) error {
	if m.Level.ID != "test-boss" || m.Finished || m.UpgradePhase != nil {
		return ErrDebugLevelUp
	}
	m.TeamLevel++
	m.beginUpgradePhase(now, "level_up")
	return nil
}
