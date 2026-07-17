package simulation

import (
	"strings"
	"time"

	"survive-bro/apps/backend/internal/protocol"
)

func (m *Match) beginUpgradePhase(now time.Time, source string, spellPools ...[]string) {
	var spellPool []string
	if len(spellPools) > 0 {
		spellPool = append([]string(nil), spellPools[0]...)
	}
	m.nextUpgradeOfferID++
	phase := &UpgradePhase{
		ID:        m.nextUpgradeOfferID,
		Source:    source,
		Deadline:  now.Add(UpgradeSelectionTimeout),
		Offers:    make(map[string]*PlayerUpgradeOffer, len(m.Players)),
		SpellPool: spellPool,
	}
	for _, playerID := range sortedPlayerIDs(m.Players) {
		player := m.Players[playerID]
		phase.Offers[playerID] = m.createPlayerOffer(player, source, spellPool)
		player.MoveX, player.MoveY = 0, 0
	}
	m.UpgradePhase = phase
}

func (m *Match) createPlayerOffer(player *Player, source string, spellPools ...[]string) *PlayerUpgradeOffer {
	var spellPool []string
	if len(spellPools) > 0 {
		spellPool = spellPools[0]
	}
	choices := m.createUpgradeChoices(player, source, spellPool)
	if source == "spell_chest" && len(choices) == 0 {
		source = "treasure_chest"
		choices = m.createUpgradeChoices(player, source, nil)
	}
	return &PlayerUpgradeOffer{Source: source, Choices: choices, SelectedIndex: -1}
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

func (m *Match) createUpgradeChoices(player *Player, source string, spellPool []string) []protocol.UpgradeChoice {
	if source == "spell_chest" {
		if len(player.SpellIDs) >= maximumPlayerSpells {
			return nil
		}
		available := make([]string, 0, len(spellPool))
		for _, spellID := range spellPool {
			_, ok := SpellByID(spellID)
			if !ok || player.SpellLevels[spellID] > 0 {
				continue
			}
			available = append(available, spellID)
		}
		m.rng.Shuffle(len(available), func(i, j int) { available[i], available[j] = available[j], available[i] })
		choices := make([]protocol.UpgradeChoice, 0, min(3, len(available)))
		for _, spellID := range available[:min(3, len(available))] {
			choices = append(choices, protocol.UpgradeChoice{Attribute: "spell:" + spellID, AddedValue: 1, FinalValue: 1})
		}
		return choices
	}
	available := append([]string(nil), availableUpgradeAttributes(player)...)
	m.rng.Shuffle(len(available), func(i, j int) { available[i], available[j] = available[j], available[i] })
	choices := make([]protocol.UpgradeChoice, 0, 3)
	for _, attribute := range available[:min(3, len(available))] {
		if spellID, ok := spellLevelAttribute(attribute); ok {
			current := player.SpellLevels[spellID]
			choices = append(choices, protocol.UpgradeChoice{Attribute: attribute, CurrentValue: float64(current), AddedValue: 1, FinalValue: float64(current + 1)})
			continue
		}
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
	if spellID, ok := spellLevelAttribute(attribute); ok {
		return float64(player.SpellLevels[spellID])
	}
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

func spellLevelAttribute(attribute string) (string, bool) {
	const prefix, suffix = "spell:", ":level"
	if !strings.HasPrefix(attribute, prefix) || !strings.HasSuffix(attribute, suffix) {
		return "", false
	}
	spellID := strings.TrimSuffix(strings.TrimPrefix(attribute, prefix), suffix)
	return spellID, spellID != ""
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
		Source:       offer.Source,
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
		attribute := offer.Choices[choiceIndex].Attribute
		if offer.Source == "spell_chest" && strings.HasPrefix(attribute, "spell:") {
			events.AppliedUpgrades = append(events.AppliedUpgrades, m.applySpellChoice(m.Players[playerID], strings.TrimPrefix(attribute, "spell:")))
			continue
		}
		events.AppliedUpgrades = append(events.AppliedUpgrades, m.applyUpgrade(m.Players[playerID], offer.Source, attribute))
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
