# Inventory and Modifier Model

The canonical content source is `game-data/game.json`. Character, spell, enemy, level, spell-slot, and spell-level modifier sections are loaded directly at server startup. Unique spell acquisition, authoritative per-spell snapshots, deterministic spell-level rewards, and clickable HUD spell details are runtime. Buff inventory and full generic modifier evaluation remain pending.

## Inventory

Each player owns two independent collections:

- Spell inventory: maximum entries are configured by `inventory.spellSlots`, currently `4`.
- Buff inventory: maximum 5 entries.

An inventory entry is `{ id, level }`. Level-up and treasure rewards use the same eligibility rules. Both are team-wide triggers: every player independently receives one eligible reward when the team levels up or any living player collects a treasure chest.

1. An unowned item is eligible only when its inventory has an empty slot.
2. Spell chests exclude every owned spell; duplicate spell entries are invalid.
3. Selecting an unowned spell adds it at level 1 without disabling existing spells.
4. Every owned spell auto-casts on its own cooldown; Aura is continuously active instead.
5. Chest pools are event-configured as `all` or explicit IDs and roll three random unowned spells. A chest with no eligible spell becomes a treasure offer for that player.
6. Buff and future general-reward leveling rules remain pending. Never silently replace an owned item.

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

## Laboratory spell progression

Heavy Aura starts as a `110`-unit field dealing `4` every `500 ms`. Levels add radius `+30`, damage `+2`, damage interval `-75 ms`, radius `+40`, damage `+4`, then damage interval `-75 ms`.

Meteorite starts by marking the nearest enemy within `700` units for `900 ms`, then dealing `42` in an `85`-unit area. Levels add radius `+20`, damage `+16`, warning `-200 ms`, cooldown `-400 ms`, radius `+25`, damage `+24`, then a second nearest-enemy meteor.

Tracking Beam starts as a `600 x 18` piercing channel lasting `1200 ms` and dealing `6` every `250 ms`. Levels add length `+100`, damage `+3`, duration `+400 ms`, width `+8`, damage interval `-50 ms`, cooldown `-300 ms`, then a second independently tracked beam.
