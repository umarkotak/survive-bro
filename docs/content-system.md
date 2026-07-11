# Game Content System

This is the central extension model for gameplay content. `game-data/game.json` is the single editable source. Sections listed in `runtimeSections` are read directly, decoded with Sonic, validated, and loaded at backend startup. Other sections remain a design contract until their gameplay systems are implemented.

## Stable content boundaries

### Character

A character owns a stable ID, display name, sprite-set ID, player attributes, starting spell IDs, and one required default spell ID. `GET /api/v1/characters` drives selection. `join_room.characterId` selects the server definition. Snapshots repeat `characterId` so every client chooses the correct sprite set.

### Spell

A spell is reusable and independent from characters. `projectile` spells use speed/range/radius; `beam` spells use length/width/linger/damage interval; `explosive_projectile` spells add direct impact damage, blast radius, and explosion linger. Characters reference starting spells and a default active spell. The player model stores multiple owned spell IDs, while acquisition and active-spell switching remain a later inventory milestone.

### Buff

A buff is an inventory item that references reusable modifiers affecting the player, one spell, matching spells, or their projectiles. Buffs and spells have independent five-slot inventories and deterministic levels.

### Modifier

A modifier targets one glossary attribute through a stable operation. The same engine is shared by spell levels, buffs, and future artifacts.

### Enemy

An enemy owns a stable ID, name, sprite ID, score, XP drop, health, flat armor, speed, collision radius, contact damage/cooldown, and optional reusable spell IDs. Enemy armor subtracts from every authoritative incoming hit with minimum damage `1`. Enemy spells use the same definitions as player spells while targeting and damage remain server-owned. Snapshots carry `typeId`; clients use that ID only for visuals.

### Level

A level owns its ID, name, duration, terrain asset IDs, obstacle asset IDs/layout, and ordered system events. `GET /api/v1/levels` drives room creation. A room locks its selected level for its lifetime.

### System event

Every event has a stable ID, timestamp, type, title, and player-facing description. The initial supported types are:

- `spawn_rate`: replaces the active normal-spawn configuration. It independently controls rate per second, maximum living enemies, and weighted enemy composition.
- `monster_buff`: multiplies health and movement speed for existing enemies and all enemies spawned afterward.
- `meteor_shower`: spawns bounded, server-authoritative area hazards near living players. Its content config owns duration, spawn rate, warning, radius, damage interval, and a validated 3–4 second linger range; snapshots expose only rendering state.
- `boss`: spawns one enemy type without changing normal spawns. Optional `endMatchOnDeath` binds victory to that exact spawned boss instance. Optional positive `statMultipliers` independently scale health, movement speed, attack damage, collision radius, contact cooldown, XP drop, and score. Omitted multipliers equal `1`; boss multipliers compose with earlier global monster buffs.
- Non-ending boss-event enemies guarantee one power crate when defeated; ordinary enemies retain the global kill-cadence crate reward. An ending boss resolves results immediately instead of dropping an unusable post-match crate.
- `end`: resolves the match and score.

Events must be ordered and deterministic. Adding an event type requires validation, simulation handling, timeline presentation, protocol documentation, and a manual acceptance case. `match_started` sends the public event timeline once; snapshots only advance remaining time.

## Extension checklist

1. Add the server definition and stable IDs.
2. Add matching client sprite assets and sprite-set mapping.
3. Expose only selection-safe metadata through the content endpoints.
4. Update binary contracts before server/client payload changes.
5. Update `mvp-spec.md`, `levels.md`, and the asset manifest when player-visible behavior changes.
