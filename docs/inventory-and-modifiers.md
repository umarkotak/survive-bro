# Inventory and Modifier Model

The canonical content source is `game-data/game.json`. Character, spell, enemy, and level sections are loaded directly at server startup. Inventory, buffs, and modifier evaluation remain design contracts until the inventory milestone replaces direct random upgrades.

## Inventory

Each player owns two independent collections:

- Spell inventory: maximum 5 entries.
- Buff inventory: maximum 5 entries.

An inventory entry is `{ id, level }`. Level-up and treasure rewards use the same eligibility rules. Both are team-wide triggers: every player independently receives one eligible reward when the team levels up or any living player collects a treasure chest.

1. An unowned item is eligible only when its inventory has an empty slot.
2. An owned item is eligible while below its maximum level.
3. Selecting an unowned item adds it at level 1.
4. Selecting an owned item increases it exactly one level.
5. Full inventories exclude unowned items; max-level entries are also excluded.
6. If no item is eligible, no reward is applied. Never silently replace an equipped item.

The server owns inventory, rolls, levels, resolved modifiers, and combat. The client displays offers/history and predicts visuals only.

## Common content envelope

Every selectable or addressable content record should provide:

- Stable lowercase ID used in contracts and references.
- Player-facing name and description.
- Kind and optional tags.
- Asset references, never raw URLs in gameplay rules.
- Base attributes using glossary attribute IDs.
- Progression with a maximum level and modifier references.

## Modifiers

A modifier is the reusable unit of effect:

```text
target + optional selector + attribute + operation + value
```

Supported initial operations:

- `add_flat`
- `add_percent_of_base`
- `multiply`
- `set`
- `clamp_max`

Evaluation order is `set`, flat additions, percent-of-base additions, multiplication, then clamps. Within one phase, sort by stable modifier ID. This makes results deterministic regardless of inventory insertion order.

Buffs reference player or spell modifiers. Spell levels reference spell/projectile modifiers. Future artifacts use the same modifier format and do not require another effect engine.

## Fireball progression

Fireball is the initial spell and begins in Ranger’s spell inventory at level 1. Its level progression is deterministic:

| Level | Change |
| ---: | --- |
| 1 | Base Fireball. |
| 2 | Damage `+4`. |
| 3 | Projectile speed `+70`. |
| 4 | Directions `+1`. |
| 5 | Burst `+1`. |

This replaces ambiguous random sub-stat upgrades. A Fireball at the same level always has the same spell-level effects; buffs and future artifacts may further modify its resolved values.

## Soul Track progression

Frieren starts with Soul Track level 1. It is a lingering beam rather than a moving projectile.

| Level | Change |
| ---: | --- |
| 1 | Base beam: `520 × 32`, 1 second linger, 1.5 second cooldown, 18 damage every 0.5 second. |
| 2 | Length `+100`. |
| 3 | Cooldown `-150 ms`. |
| 4 | Linger `+250 ms`. |
| 5 | Width `+10`. |
| 6 | Directions `+1`. |
| 7 | Damage `+6`. |

## Rocket progression

Catapult starts with Rocket level 1. Rocket remains an independent spell that may later be acquired by another character.

| Level | Change |
| ---: | --- |
| 1 | 20 impact damage, 30 explosion-tick damage, 480 speed, 850 range, 80 blast radius, 1 second linger, 0.5 second damage interval, 1.6 second cooldown. |
| 2 | Damage `+8`. |
| 3 | Explosion radius `+20`. |
| 4 | Explosion linger `+250 ms`. |
| 5 | Cooldown `-150 ms`. |
| 6 | Projectile speed `+60`. |
| 7 | Directions `+1`. |

## Runtime migration checkpoint

Base character, spell, enemy, and level content is already loaded from JSON at startup. Before inventory gameplay is enabled:

1. Extend startup validation to modifiers, buffs, progression references, caps, and progression gaps.
2. Decode modifiers, buffs, and spell-level progression into immutable runtime data.
3. Add binary inventory snapshots and authoritative reward-applied events.
4. Replace the existing direct random attribute upgrade switch with inventory add/level operations.
5. Add the 5-spell/5-buff inventory UI and manual acceptance gate.

Keeping this migration explicit prevents two simultaneous sources of gameplay truth.
