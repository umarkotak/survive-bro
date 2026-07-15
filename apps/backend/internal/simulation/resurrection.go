package simulation

import (
	"math"
	"sort"
)

// applyPlayerDamage is the single authority for immunity and death transitions.
func (m *Match) applyPlayerDamage(player *Player, damage int) {
	if !player.Alive || m.Elapsed < player.ImmuneUntil {
		return
	}
	player.HP = max(0, player.HP-damage)
	if player.HP > 0 {
		return
	}
	player.Alive = false
	player.MoveX, player.MoveY = 0, 0
	player.VelocityX, player.VelocityY = 0, 0
	player.ResurrectionProgress = 0
	m.assignAvailableLives()
}

// assignAvailableLives reserves pooled team lives for dead players in stable ID order.
func (m *Match) assignAvailableLives() {
	ids := make([]string, 0, len(m.Players))
	for id := range m.Players {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		player := m.Players[id]
		if m.availableTeamLives() == 0 {
			return
		}
		if player.Alive || player.ResurrectionPending {
			continue
		}
		player.ResurrectionPending = true
		player.ResurrectionProgress = 0
	}
}

func (m *Match) updateResurrections() {
	for _, id := range sortedPlayerIDs(m.Players) {
		player := m.Players[id]
		if player.Alive || !player.ResurrectionPending {
			continue
		}
		if len(m.Players) > 1 && !m.hasReviverWithin(player) {
			player.ResurrectionProgress = 0
			continue
		}
		player.ResurrectionProgress += TickDuration
		if player.ResurrectionProgress < player.ResurrectionDuration {
			continue
		}
		player.Alive = true
		player.HP = max(1, int(math.Ceil(float64(player.MaxHP)*0.5)))
		m.TeamLives = max(0, m.TeamLives-1)
		player.ResurrectionPending = false
		player.ResurrectionProgress = 0
		player.ImmuneUntil = m.Elapsed + player.ResurrectionImmunityDuration
		player.LastAttackAt = m.Elapsed
	}
}

func (m *Match) availableTeamLives() int {
	reserved := 0
	for _, player := range m.Players {
		if player.ResurrectionPending {
			reserved++
		}
	}
	return max(0, m.TeamLives-reserved)
}

func (m *Match) hasReviverWithin(dead *Player) bool {
	for _, player := range m.Players {
		if player.ID != dead.ID && player.Alive && math.Hypot(player.X-dead.X, player.Y-dead.Y) <= dead.ResurrectionRadius {
			return true
		}
	}
	return false
}

func (m *Match) hasSoloResurrection() bool {
	if len(m.Players) != 1 {
		return false
	}
	for _, player := range m.Players {
		return player.ResurrectionPending
	}
	return false
}

func resurrectionProgress(player *Player) float64 {
	if !player.ResurrectionPending || player.ResurrectionDuration <= 0 {
		return 0
	}
	return min(1, float64(player.ResurrectionProgress)/float64(player.ResurrectionDuration))
}
