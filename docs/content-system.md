# Game Content System

This is the central extension model for gameplay content. `game-data/game.json` is the single editable source. Sections listed in `runtimeSections` are read directly, decoded with Sonic, validated, and loaded at backend startup. Other sections remain a design contract until their gameplay systems are implemented.

## Stable content boundaries

### Character

A character owns a stable ID, display name, sprite-set ID, player attributes, starting spell IDs, and one required default spell ID. `GET /api/v1/characters` drives selection. `join_room.characterId` selects the server definition. Snapshots repeat `characterId` so every client chooses the correct sprite set.

### Spell

A spell is reusable and independent from characters. Player-available spells may appear in `all` spell-chest pools; enemy-only spells cannot. `projectile` spells use speed/range/radius; `beam` spells use length/width/linger/damage interval; `tracking_beam` continuously retargets during a channel; `explosive_projectile` adds impact and blast damage; `aura` is always present while its living owner holds it and damages enemies immediately when they enter; `player_meteor` marks the nearest enemy position before an area impact; and `melee` applies an immediate hit inside configured range.

### Buff

A buff is an inventory item that references reusable modifiers affecting the player, one spell, matching spells, or their projectiles. Buffs and spells have independent five-slot inventories and deterministic levels.

### Modifier

A modifier targets one glossary attribute through a stable operation. The same engine is shared by spell levels, buffs, and future artifacts.

### Enemy

An enemy owns a stable ID, name, sprite ID, score, XP drop, health, flat armor, speed, collision radius, contact damage/cooldown, and optional reusable spell IDs. Enemy armor subtracts from every authoritative incoming hit with minimum damage `1`. Enemy spells use the same definitions as player spells, including damage, cooldown, range, projectile geometry, and effect asset IDs, while targeting and damage remain server-owned. Cooldown state is tracked independently per owned spell so adding another spell does not block or reset the others. Stage 1 Slimes and Slime Kings reference the near-contact `slime-punch` melee spell; Greater Slimes reference `enemy-slime-ball`, a short-range projectile independent from player Fireball. Snapshots carry `typeId`; clients use that ID only for visuals.

Runtime content may include clearly named dummy characters and levels for manual development. `dummy-tester` reuses the Ranger sprite set with durable test stats; `test-boss` spawns a high-health Slime King as soon as the first player exists. These entries use the same selection endpoints and validation as production content.

`spell-lab` is a bounded test level for spell acquisition and upgrades. Its nine `all` spell chests draw from Fireball, Soul Track, Rocket, Heavy Aura, Meteorite, and Tracking Beam, roll three random unowned choices, and become treasure offers after the four-spell inventory is full. Every owned spell auto-casts independently, while normal rewards may level a specifically named owned spell.

### Level

A level owns its ID, name, duration, terrain asset IDs, obstacle asset IDs/layout, and ordered system events. `GET /api/v1/levels` drives room creation. A room locks its selected level for its lifetime.

### System event

Every event has a stable ID, timestamp, type, title, and player-facing description. The initial supported types are:

- `spawn_rate`: replaces the active normal-spawn configuration. It independently controls rate per second, maximum living enemies, and weighted enemy composition.
- `monster_buff`: multiplies health and movement speed for existing enemies and all enemies spawned afterward.
- `meteor_shower`: spawns bounded, server-authoritative area hazards near living players. Its content config owns duration, spawn rate, warning, radius, damage interval, and a validated 3–4 second linger range; snapshots expose only rendering state.
- `treasure_rate`: changes normal power-crate drops to one per configured number of kills. Guaranteed boss crates remain independent.
- `spell_chest`: spawns one spell chest at a validated world position. Its config uses either `spellPool: "all"` or a unique explicit `spellIds` list. Collection rolls up to three random unowned player-available spells and opens the synchronized offer phase.
- `boss`: spawns one enemy type without changing normal spawns. Optional `endMatchOnDeath` binds victory to that exact spawned boss instance. Optional positive `statMultipliers` independently scale health, movement speed, attack damage, collision radius, contact cooldown, XP drop, and score. Omitted multipliers equal `1`; boss multipliers compose with earlier global monster buffs.
- Non-ending boss-event enemies guarantee one power crate when defeated; ordinary enemies retain the global kill-cadence crate reward. An ending boss resolves results immediately instead of dropping an unusable post-match crate.
- `end`: resolves the match and score.

Events must be ordered and deterministic. Each event has a `show` flag; omitted values default to visible, while hidden events execute normally but are omitted from `match_started`. Adding an event type requires validation, simulation handling, protocol documentation, and a manual acceptance case. Snapshots only advance remaining time.

## Extension checklist

1. Add the server definition and stable IDs.
2. Add matching client sprite assets and sprite-set mapping.
3. Expose only selection-safe metadata through the content endpoints.
4. Update binary contracts before server/client payload changes.
5. Update `mvp-spec.md`, `levels.md`, and the asset manifest when player-visible behavior changes.
