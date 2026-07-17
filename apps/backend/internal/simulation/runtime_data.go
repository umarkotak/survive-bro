package simulation

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/bytedance/sonic"
)

type runtimeGameData struct {
	Inventory struct {
		SpellSlots int `json:"spellSlots"`
	} `json:"inventory"`
	Modifiers map[string]struct {
		Target   string `json:"target"`
		Selector struct {
			SpellID string `json:"spellId"`
		} `json:"selector"`
		Attribute string  `json:"attribute"`
		Operation string  `json:"operation"`
		Value     float64 `json:"value"`
	} `json:"modifiers"`
	Spells map[string]struct {
		AttackType      string `json:"attackType"`
		PlayerAvailable bool   `json:"playerAvailable"`
		BaseAttributes  struct {
			Damage           int     `json:"damage"`
			ImpactDamage     int     `json:"impact_damage"`
			CooldownMS       int64   `json:"cooldown_ms"`
			ProjectileSpeed  float64 `json:"projectile_speed"`
			ProjectileRange  float64 `json:"projectile_range"`
			ProjectileRadius float64 `json:"projectile_radius"`
			MeleeRange       float64 `json:"melee_range"`
			Burst            int     `json:"burst"`
			Directions       int     `json:"directions"`
			BeamLength       float64 `json:"beam_length"`
			BeamWidth        float64 `json:"beam_width"`
			LingerDurationMS int64   `json:"linger_duration_ms"`
			DamageIntervalMS int64   `json:"damage_interval_ms"`
			ExplosionRadius  float64 `json:"explosion_radius"`
		} `json:"baseAttributes"`
		MaxLevel int                 `json:"maxLevel"`
		Levels   map[string][]string `json:"levels"`
	} `json:"spells"`
	Characters map[string]struct {
		Name           string `json:"name"`
		SpriteSet      string `json:"spriteSet"`
		DefaultSpellID string `json:"defaultSpellId"`
		BaseAttributes struct {
			MaxHealth                    int     `json:"max_health"`
			ArmorPercent                 float64 `json:"armor_percent"`
			MovementSpeed                float64 `json:"movement_speed"`
			HealthRegeneration           float64 `json:"health_regeneration"`
			AttackBuffPercent            float64 `json:"attack_buff_percent"`
			CooldownReduction            float64 `json:"cooldown_reduction_percent"`
			PickupRadius                 float64 `json:"pickup_radius"`
			ResurrectionDuration         float64 `json:"resurrection_duration"`
			ResurrectionRadius           float64 `json:"resurrection_radius"`
			ResurrectionImmunityDuration float64 `json:"resurrection_immunity_duration"`
		} `json:"baseAttributes"`
		StartingInventory struct {
			Spells []struct {
				ID string `json:"id"`
			} `json:"spells"`
		} `json:"startingInventory"`
	} `json:"characters"`
	Enemies map[string]struct {
		Name       string   `json:"name"`
		Sprite     string   `json:"sprite"`
		SpellIDs   []string `json:"spellIds"`
		Attributes struct {
			Score           int     `json:"score"`
			Experience      int     `json:"experience_drop"`
			Health          int     `json:"health"`
			MovementSpeed   float64 `json:"movement_speed"`
			Damage          int     `json:"damage"`
			Armor           int     `json:"armor"`
			CooldownMS      int64   `json:"cooldown_ms"`
			CollisionRadius float64 `json:"collision_radius"`
		} `json:"attributes"`
	} `json:"enemies"`
	Levels map[string]runtimeLevel `json:"levels"`
}

type runtimeLevel struct {
	Name           string         `json:"name"`
	DurationMS     int64          `json:"duration_ms"`
	TerrainAssets  []string       `json:"terrainAssets"`
	ObstacleAssets []string       `json:"obstacleAssets"`
	Events         []runtimeEvent `json:"events"`
}

type runtimeEvent struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	AtMS        int64  `json:"at_ms"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Show        *bool  `json:"show"`
	Config      struct {
		SpawnRate        float64  `json:"spawn_rate"`
		MaxLiving        int      `json:"max_living"`
		HealthMultiplier float64  `json:"health_multiplier"`
		SpeedMultiplier  float64  `json:"speed_multiplier"`
		DurationMS       int64    `json:"duration_ms"`
		RatePerSecond    float64  `json:"rate_per_second"`
		WarningMS        int64    `json:"warning_ms"`
		LingerMinMS      int64    `json:"linger_min_ms"`
		LingerMaxMS      int64    `json:"linger_max_ms"`
		Radius           float64  `json:"radius"`
		X                float64  `json:"x"`
		Y                float64  `json:"y"`
		Damage           int      `json:"damage"`
		DamageIntervalMS int64    `json:"damage_interval_ms"`
		KillsPerChest    int      `json:"kills_per_chest"`
		SpellPool        string   `json:"spellPool"`
		SpellIDs         []string `json:"spellIds"`
		EnemyID          string   `json:"enemyId"`
		Count            int      `json:"count"`
		EndMatchOnDeath  bool     `json:"endMatchOnDeath"`
		Composition      []struct {
			EnemyID string `json:"enemyId"`
			Weight  int    `json:"weight"`
		} `json:"composition"`
		StatMultipliers struct {
			Health          float64 `json:"health"`
			MovementSpeed   float64 `json:"movement_speed"`
			Damage          float64 `json:"damage"`
			CollisionRadius float64 `json:"collision_radius"`
			ContactCooldown float64 `json:"cooldown_ms"`
			ExperienceDrop  float64 `json:"experience_drop"`
			Score           float64 `json:"score"`
		} `json:"statMultipliers"`
	} `json:"config"`
}

// LoadRuntimeGameData replaces runtime character, spell, enemy, and level definitions at startup.
// Existing rooms retain the immutable definitions copied when they were created.
func LoadRuntimeGameData(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read game data %s: %w", path, err)
	}
	var source runtimeGameData
	if err := sonic.Unmarshal(data, &source); err != nil {
		return fmt.Errorf("decode game data %s: %w", path, err)
	}
	if source.Inventory.SpellSlots < 1 || source.Inventory.SpellSlots > 8 {
		return fmt.Errorf("inventory spellSlots must be between 1 and 8")
	}
	loadedSpells := make(map[string]SpellDefinition, len(source.Spells))
	for id, value := range source.Spells {
		a := value.BaseAttributes
		kind := value.AttackType
		if kind == "" {
			kind = "projectile"
		}
		if a.Burst == 0 {
			a.Burst = 1
		}
		if a.Directions == 0 {
			a.Directions = 1
		}
		if id == "" || a.Damage <= 0 || a.CooldownMS <= 0 || a.Directions < 1 || a.Directions > 4 || a.Burst < 1 || a.Burst > 2 {
			return fmt.Errorf("spell %q has invalid required values", id)
		}
		spell := SpellDefinition{ID: id, Kind: kind, Damage: a.Damage, ImpactDamage: a.ImpactDamage, Cooldown: time.Duration(a.CooldownMS) * time.Millisecond, ProjectileSpeed: a.ProjectileSpeed, Range: a.ProjectileRange, Radius: a.ProjectileRadius, Burst: a.Burst, Directions: a.Directions, BeamLength: a.BeamLength, BeamWidth: a.BeamWidth, Duration: time.Duration(a.LingerDurationMS) * time.Millisecond, DamageInterval: time.Duration(a.DamageIntervalMS) * time.Millisecond, ExplosionRadius: a.ExplosionRadius, ExplosionDuration: time.Duration(a.LingerDurationMS) * time.Millisecond, MaxLevel: value.MaxLevel, Levels: make(map[int][]SpellModifier), PlayerAvailable: value.PlayerAvailable}
		if kind != "projectile" && kind != "explosive_projectile" && kind != "beam" && kind != "tracking_beam" && kind != "aura" && kind != "player_meteor" && kind != "melee" {
			return fmt.Errorf("spell %q has unsupported attack type %q", id, kind)
		}
		if kind == "beam" || kind == "tracking_beam" {
			spell.Range = spell.BeamLength
		}
		if kind == "melee" {
			spell.Range = a.MeleeRange
		}
		if kind == "aura" {
			spell.Range = spell.ExplosionRadius
		}
		if kind == "projectile" || kind == "explosive_projectile" {
			if spell.ProjectileSpeed <= 0 || spell.Range <= 0 || spell.Radius <= 0 {
				return fmt.Errorf("spell %q has invalid projectile values", id)
			}
		}
		if (kind == "beam" || kind == "tracking_beam") && (spell.BeamLength <= 0 || spell.BeamWidth <= 0 || spell.Duration <= 0 || spell.DamageInterval <= 0) {
			return fmt.Errorf("spell %q has invalid beam values", id)
		}
		if kind == "aura" && (spell.ExplosionRadius <= 0 || spell.Duration <= 0 || spell.DamageInterval <= 0) {
			return fmt.Errorf("spell %q has invalid aura values", id)
		}
		if kind == "player_meteor" && (spell.Range <= 0 || spell.ExplosionRadius <= 0 || spell.Duration <= 0) {
			return fmt.Errorf("spell %q has invalid meteor values", id)
		}
		if kind == "melee" && spell.Range <= 0 {
			return fmt.Errorf("spell %q has invalid melee range", id)
		}
		if kind == "explosive_projectile" && (spell.ExplosionRadius <= 0 || spell.ExplosionDuration <= 0 || spell.DamageInterval <= 0) {
			return fmt.Errorf("spell %q has invalid explosion values", id)
		}
		if spell.MaxLevel < 1 {
			return fmt.Errorf("spell %q has invalid max level", id)
		}
		for levelText, modifierIDs := range value.Levels {
			level, err := strconv.Atoi(levelText)
			if err != nil || level < 2 || level > spell.MaxLevel {
				return fmt.Errorf("spell %q has invalid level %q", id, levelText)
			}
			for _, modifierID := range modifierIDs {
				modifier, exists := source.Modifiers[modifierID]
				if !exists || modifier.Target != "spell" || modifier.Selector.SpellID != id || modifier.Operation != "add_flat" {
					return fmt.Errorf("spell %q has invalid modifier %q", id, modifierID)
				}
				spell.Levels[level] = append(spell.Levels[level], SpellModifier{Attribute: modifier.Attribute, Value: modifier.Value})
			}
		}
		loadedSpells[id] = spell
	}
	loadedCharacters := make(map[string]CharacterDefinition, len(source.Characters))
	for id, value := range source.Characters {
		a := value.BaseAttributes
		if id == "" || value.Name == "" || value.SpriteSet == "" || a.MaxHealth <= 0 || a.MovementSpeed <= 0 || a.PickupRadius <= 0 || a.ResurrectionDuration <= 0 || a.ResurrectionRadius <= 0 || a.ResurrectionImmunityDuration <= 0 {
			return fmt.Errorf("character %q has invalid required values", id)
		}
		if _, exists := loadedSpells[value.DefaultSpellID]; !exists {
			return fmt.Errorf("character %q references unknown default spell", id)
		}
		if !loadedSpells[value.DefaultSpellID].PlayerAvailable {
			return fmt.Errorf("character %q default spell is not player available", id)
		}
		spellIDs := make([]string, 0, len(value.StartingInventory.Spells))
		seenSpellIDs := make(map[string]struct{}, len(value.StartingInventory.Spells))
		for _, entry := range value.StartingInventory.Spells {
			if _, exists := loadedSpells[entry.ID]; !exists {
				return fmt.Errorf("character %q references unknown starting spell", id)
			}
			if _, duplicate := seenSpellIDs[entry.ID]; duplicate {
				return fmt.Errorf("character %q has duplicate starting spell %q", id, entry.ID)
			}
			seenSpellIDs[entry.ID] = struct{}{}
			spellIDs = append(spellIDs, entry.ID)
		}
		if len(spellIDs) == 0 || len(spellIDs) > source.Inventory.SpellSlots {
			return fmt.Errorf("character %q has no starting spell", id)
		}
		loadedCharacters[id] = CharacterDefinition{ID: id, Name: value.Name, SpriteID: value.SpriteSet, MaxHP: a.MaxHealth, ArmorPercent: a.ArmorPercent, MovementSpeed: a.MovementSpeed, HealthRegeneration: a.HealthRegeneration, AttackBuffPercent: a.AttackBuffPercent, CooldownPercent: a.CooldownReduction, PickupRadius: a.PickupRadius, ResurrectionDuration: time.Duration(a.ResurrectionDuration * float64(time.Second)), ResurrectionRadius: a.ResurrectionRadius, ResurrectionImmunityDuration: time.Duration(a.ResurrectionImmunityDuration * float64(time.Second)), DefaultSpellID: value.DefaultSpellID, StartingSpellIDs: spellIDs}
	}
	loadedEnemies := make(map[string]EnemyDefinition, len(source.Enemies))
	for id, value := range source.Enemies {
		a := value.Attributes
		if a.Armor == 0 {
			a.Armor = 1
		}
		if id == "" || value.Name == "" || value.Sprite == "" || a.Health <= 0 || a.Armor < 0 || a.MovementSpeed <= 0 || a.Damage <= 0 || a.CooldownMS <= 0 || a.CollisionRadius <= 0 || a.Experience <= 0 || a.Score <= 0 {
			return fmt.Errorf("enemy %q has invalid required values", id)
		}
		for _, spellID := range value.SpellIDs {
			if _, exists := loadedSpells[spellID]; !exists {
				return fmt.Errorf("enemy %q references unknown spell %q", id, spellID)
			}
		}
		loadedEnemies[id] = EnemyDefinition{ID: id, Name: value.Name, SpriteID: value.Sprite, Score: a.Score, Experience: a.Experience, MaxHP: a.Health, Speed: a.MovementSpeed, Radius: a.CollisionRadius, ContactDamage: a.Damage, Armor: a.Armor, SpellIDs: append([]string(nil), value.SpellIDs...), ContactDelay: time.Duration(a.CooldownMS) * time.Millisecond}
	}
	loadedLevels := make(map[string]LevelDefinition, len(source.Levels))
	for id, level := range source.Levels {
		if id == "" || level.DurationMS <= 0 || level.Name == "" {
			return fmt.Errorf("level %q is invalid", id)
		}
		events := make([]LevelEvent, 0, len(level.Events))
		sort.SliceStable(level.Events, func(i, j int) bool { return level.Events[i].AtMS < level.Events[j].AtMS })
		previousAt := int64(-1)
		eventIDs := make(map[string]struct{}, len(level.Events))
		for _, value := range level.Events {
			if value.ID == "" || value.AtMS < previousAt || value.AtMS > level.DurationMS {
				return fmt.Errorf("level %q event %q has invalid ordering or time", id, value.ID)
			}
			if _, duplicate := eventIDs[value.ID]; duplicate {
				return fmt.Errorf("level %q has duplicate event %q", id, value.ID)
			}
			eventIDs[value.ID] = struct{}{}
			previousAt = value.AtMS
			show := true
			if value.Show != nil {
				show = *value.Show
			}
			event := LevelEvent{ID: value.ID, At: time.Duration(value.AtMS) * time.Millisecond, Type: value.Type, Title: value.Title, Description: value.Description, Show: show, EnemyID: value.Config.EnemyID, EndMatchOnDeath: value.Config.EndMatchOnDeath, BossCount: max(1, value.Config.Count)}
			switch value.Type {
			case "spawn_rate":
				if value.Config.SpawnRate <= 0 || value.Config.MaxLiving <= 0 || len(value.Config.Composition) == 0 {
					return fmt.Errorf("event %q has invalid spawn rate values", value.ID)
				}
				spawn := &SpawnRateDefinition{RatePerSecond: value.Config.SpawnRate, MaxLiving: value.Config.MaxLiving}
				for _, entry := range value.Config.Composition {
					if _, exists := loadedEnemies[entry.EnemyID]; !exists || entry.Weight <= 0 {
						return fmt.Errorf("event %q has invalid spawn entry", value.ID)
					}
					spawn.Entries = append(spawn.Entries, SpawnEntry{EnemyID: entry.EnemyID, Weight: entry.Weight})
				}
				event.SpawnRate = spawn
			case "monster_buff":
				if value.Config.HealthMultiplier <= 0 || value.Config.SpeedMultiplier <= 0 {
					return fmt.Errorf("event %q has invalid monster buff values", value.ID)
				}
				event.MonsterBuff = &MonsterBuffDefinition{HealthMultiplier: value.Config.HealthMultiplier, SpeedMultiplier: value.Config.SpeedMultiplier}
			case "meteor_shower":
				c := value.Config
				if c.DurationMS <= 0 || c.RatePerSecond <= 0 || c.WarningMS <= 0 || c.LingerMinMS < 3000 || c.LingerMaxMS < c.LingerMinMS || c.Radius <= 0 || c.Damage <= 0 || c.DamageIntervalMS <= 0 {
					return fmt.Errorf("event %q has invalid meteor shower values", value.ID)
				}
				event.MeteorShower = &MeteorShowerDefinition{Duration: time.Duration(c.DurationMS) * time.Millisecond, RatePerSecond: c.RatePerSecond, Warning: time.Duration(c.WarningMS) * time.Millisecond, LingerMin: time.Duration(c.LingerMinMS) * time.Millisecond, LingerMax: time.Duration(c.LingerMaxMS) * time.Millisecond, Radius: c.Radius, Damage: c.Damage, DamageInterval: time.Duration(c.DamageIntervalMS) * time.Millisecond}
			case "spell_chest":
				if value.Config.X < 0 || value.Config.X > WorldWidth || value.Config.Y < 0 || value.Config.Y > WorldHeight {
					return fmt.Errorf("event %q has invalid spell chest position", value.ID)
				}
				pool := make([]string, 0)
				if value.Config.SpellPool == "all" {
					for spellID, spell := range loadedSpells {
						if spell.PlayerAvailable {
							pool = append(pool, spellID)
						}
					}
					sort.Strings(pool)
				} else {
					if value.Config.SpellPool != "" || len(value.Config.SpellIDs) == 0 {
						return fmt.Errorf("event %q has invalid spell pool", value.ID)
					}
					seen := make(map[string]struct{}, len(value.Config.SpellIDs))
					for _, spellID := range value.Config.SpellIDs {
						spell, exists := loadedSpells[spellID]
						if !exists || !spell.PlayerAvailable {
							return fmt.Errorf("event %q references unavailable spell %q", value.ID, spellID)
						}
						if _, duplicate := seen[spellID]; duplicate {
							return fmt.Errorf("event %q duplicates spell %q", value.ID, spellID)
						}
						seen[spellID] = struct{}{}
						pool = append(pool, spellID)
					}
				}
				event.SpellChest = &SpellChestDefinition{X: value.Config.X, Y: value.Config.Y, SpellIDs: pool}
			case "treasure_rate":
				if value.Config.KillsPerChest < 1 || value.Config.KillsPerChest > 1000 {
					return fmt.Errorf("event %q has invalid treasure cadence", value.ID)
				}
				event.TreasureRate = &TreasureRateDefinition{KillsPerChest: value.Config.KillsPerChest}
			case "boss":
				if _, exists := loadedEnemies[event.EnemyID]; !exists {
					return fmt.Errorf("event %q references unknown boss", value.ID)
				}
				m := value.Config.StatMultipliers
				event.BossMultipliers = &EnemyStatMultipliers{Health: m.Health, MovementSpeed: m.MovementSpeed, AttackDamage: m.Damage, CollisionRadius: m.CollisionRadius, ContactCooldown: m.ContactCooldown, ExperienceDrop: m.ExperienceDrop, Score: m.Score}
			case "end":
			default:
				return fmt.Errorf("event %q has unsupported type %q", value.ID, value.Type)
			}
			events = append(events, event)
		}
		if len(events) == 0 {
			return fmt.Errorf("level %q has no valid events", id)
		}
		loadedLevels[id] = LevelDefinition{ID: id, Name: level.Name, Duration: time.Duration(level.DurationMS) * time.Millisecond, TerrainAssetIDs: append([]string(nil), level.TerrainAssets...), ObstacleAssetIDs: append([]string(nil), level.ObstacleAssets...), Obstacles: Obstacles, Events: events}
	}
	loadedLevelOne, ok := loadedLevels["level-1"]
	if !ok {
		return fmt.Errorf("level-1 is missing")
	}
	spells = loadedSpells
	characters = loadedCharacters
	enemies = loadedEnemies
	levelOne = loadedLevelOne
	levelDefinitions = loadedLevels
	maximumPlayerSpells = source.Inventory.SpellSlots
	return nil
}
