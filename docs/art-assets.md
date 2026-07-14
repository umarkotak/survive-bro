# Art Asset Manifest

This document defines the production media expected by `apps/game`. All paths are relative to `apps/game`. Asset category and content ID belong in folders; filenames describe only the frame or variant inside that content folder.

The supplied terrain, three characters, Level 1 Slimes, rocks, and menu media use the hierarchy below. `src/config/assets.ts` converts stable content IDs into public paths; `BootScene` keeps stable Phaser texture keys. Generated textures remain only for assets that have not been supplied yet.

## General rules

- Use PNG.
- Terrain must be opaque and tile seamlessly on every edge.
- Characters, enemies, obstacles, bullets, pickups, effects, portraits, and icons must use transparent backgrounds.
- Do not bake shadows into characters or enemies; use the shared shadow image.
- Every character and enemy source image faces right. The game flips the image horizontally when moving left.
- Animation frames for one entity must use identical canvas dimensions, scale, foot position, and transparent padding.
- Keep important artwork inside the canvas so rotation, scaling, and hit flashes do not clip it.

## Minimal movement animation

Every moving character and enemy uses idle and walking source images. Characters with a projectile attack may also provide an attack frame:

```text
{category}/{entity-id}/idle.png
{category}/{entity-id}/walk-1.png
{category}/{entity-id}/walk-2.png
{category}/{entity-id}/walk-3.png
{category}/{entity-id}/attack-1.png
```

Behavior:

- Standing displays `idle` only.
- Ranger movement loops `walk-1 -> walk-2 -> walk-3 -> walk-2`.
- Each walking frame lasts approximately `160 ms`.
- Stopping immediately returns to `idle`.
- Vertical movement keeps the last horizontal facing direction.
- Ranger displays `attack-1` for `140 ms` whenever Arc Bolt fires, temporarily overriding idle or walking.

## Required current assets

### Terrain

| ID | Location | Canvas | Notes |
| --- | --- | ---: | --- |
| Meadow variant 1 | `public/assets/terrain/variant-1.png` | 256 x 256 | Seamless base tile. |
| Meadow variant 2 | `public/assets/terrain/variant-2.png` | 256 x 256 | Seamless visual variation. |
| Meadow variant 3 | `public/assets/terrain/variant-3.png` | 256 x 256 | Seamless visual variation. |

### Obstacles and shadows

| ID | Location | Canvas | Notes |
| --- | --- | ---: | --- |
| Large rock 1 | `public/assets/obstacle/large-rock-1.png` | 256 x 256 | Rendered at 180 x 180; visual collision remains server-owned. |
| Large rock 2 | `public/assets/obstacle/large-rock-2.png` | 256 x 256 | Same footprint as variant 1. |
| Large rock 3 | `public/assets/obstacle/large-rock-3.png` | 256 x 256 | Same footprint as variant 1. |
| Entity shadow | `public/assets/misc/entity-shadow.png` | 128 x 52 | Soft transparent ellipse; no character details. |

### Ranger

| ID | Location | Canvas |
| --- | --- | ---: |
| Idle | `public/assets/character/ranger/idle.png` | 256 x 256 |
| Walk 1 | `public/assets/character/ranger/walk-1.png` | 256 x 256 |
| Walk 2 | `public/assets/character/ranger/walk-2.png` | 256 x 256 |
| Walk 3 | `public/assets/character/ranger/walk-3.png` | 256 x 256 |
| Attack 1 | `public/assets/character/ranger/attack-1.png` | 256 x 256 |
| Selection portrait | `public/assets/character/ranger/portrait.png` | 256 x 256 |

### Frieren

| ID | Location | Canvas |
| --- | --- | ---: |
| Idle | `public/assets/character/frieren/idle.png` | 256 x 256 |
| Walk 1 | `public/assets/character/frieren/walk-1.png` | 256 x 256 |
| Walk 2 | `public/assets/character/frieren/walk-2.png` | 256 x 256 |
| Walk 3 | `public/assets/character/frieren/walk-3.png` | 256 x 256 |
| Attack 1 | `public/assets/character/frieren/attack-1.png` | 256 x 256 |

### Catapult

| ID | Location | Canvas |
| --- | --- | ---: |
| Idle | `public/assets/character/catapult/idle.png` | 256 x 256 |
| Walk 1 | `public/assets/character/catapult/walk-1.png` | 256 x 256 |
| Walk 2 | `public/assets/character/catapult/walk-2.png` | 256 x 256 |
| Walk 3 | `public/assets/character/catapult/walk-3.png` | 256 x 256 |
| Attack 1 | `public/assets/character/catapult/attack-1.png` | 256 x 256 |

### Level 1 Slimes

| ID | Location | Canvas |
| --- | --- | ---: |
| Stage 1 | `public/assets/enemy/slime-stage-1/idle.png` | 256 x 256 |
| Stage 2 | `public/assets/enemy/slime-stage-2/idle.png` | 256 x 256 |
| Stage 3 boss | `public/assets/enemy/slime-stage-3/idle.png` | 256 x 256 |

### Current attack and pickups

| ID | Location | Canvas | Notes |
| --- | --- | ---: | --- |
| Arc Bolt | `public/assets/spell/arc-bolt/projectile.png` | 48 x 24 | Points right; Phaser rotates it to trajectory. |
| Experience crystal | `public/assets/pickup/experience-crystal.png` | 28 x 28 | Centered for rotation and magnet animation. |
| Power crate | `public/assets/pickup/power-crate.png` | 48 x 48 | Gold or otherwise visually distinct from XP. |

### Menu media

| ID | Location | Notes |
| --- | --- | --- |
| Background fallback | `public/assets/misc/menu-background.png` | Always available as poster/fallback. |
| Background video | `public/assets/misc/menu-background.mp4` | Optional motion layer. |
| Transparent logo | `public/assets/misc/menu-logo.png` | Active menu logo. |
| Opaque logo source | `public/assets/misc/menu-logo-opaque.png` | Preserved alternate/source asset. |

## Planned MVP assets

These files support Guardian, the planned Brute enemy, richer feedback, and upgrade selection. They may be supplied after the required current assets.

### Guardian

| ID | Location | Canvas |
| --- | --- | ---: |
| Idle | `public/assets/character/guardian/idle.png` | 256 x 256 |
| Walk 1 | `public/assets/character/guardian/walk-1.png` | 256 x 256 |
| Walk 2 | `public/assets/character/guardian/walk-2.png` | 256 x 256 |
| Walk 3 | `public/assets/character/guardian/walk-3.png` | 256 x 256 |
| Attack 1 | `public/assets/character/guardian/attack-1.png` | 256 x 256 |
| Selection portrait | `public/assets/character/guardian/portrait.png` | 256 x 256 |
| Guardian Pulse | `public/assets/spell/guardian-pulse/activation.png` | 270 x 270 |

Guardian Pulse is a centered transparent circle. The client scales and fades it; multiple source frames are not required.

### Brute

| ID | Location | Canvas |
| --- | --- | ---: |
| Idle | `public/assets/enemy/brute/idle.png` | 256 x 256 |
| Walk 1 | `public/assets/enemy/brute/walk-1.png` | 256 x 256 |
| Walk 2 | `public/assets/enemy/brute/walk-2.png` | 256 x 256 |
| Walk 3 | `public/assets/enemy/brute/walk-3.png` | 256 x 256 |
| Attack 1 | `public/assets/enemy/brute/attack-1.png` | 256 x 256 |

### Combat and progression effects

| ID | Location | Canvas |
| --- | --- | ---: |
| Arc Bolt hit 1 | `public/assets/spell/arc-bolt/hit-1.png` | 64 x 64 |
| Arc Bolt hit 2 | `public/assets/spell/arc-bolt/hit-2.png` | 64 x 64 |
| Enemy death 1 | `public/assets/effect/enemy-death-1.png` | 96 x 96 |
| Enemy death 2 | `public/assets/effect/enemy-death-2.png` | 96 x 96 |
| Level up 1 | `public/assets/effect/level-up-1.png` | 128 x 128 |
| Level up 2 | `public/assets/effect/level-up-2.png` | 128 x 128 |
| Crystal absorption glow | `public/assets/effect/crystal-absorb.png` | 48 x 48 |
| Player damage flash | `public/assets/effect/player-damage.png` | 96 x 96 |

Two-frame effects play once and then disappear; they do not loop.

### Upgrade icons

| ID | Location | Canvas |
| --- | --- | ---: |
| Damage | `public/assets/icon/upgrade/damage.png` | 64 x 64 |
| Cooldown | `public/assets/icon/upgrade/cooldown.png` | 64 x 64 |
| Movement speed | `public/assets/icon/upgrade/movement.png` | 64 x 64 |
| Vitality | `public/assets/icon/upgrade/vitality.png` | 64 x 64 |
| Armor | `public/assets/icon/upgrade/armor.png` | 64 x 64 |
| Magnet | `public/assets/icon/upgrade/magnet.png` | 64 x 64 |

## Future naming contract

Each character owns its visual set, stats, and default attack ID. Add new characters using:

```text
public/assets/character/{character-id}/idle.png
public/assets/character/{character-id}/walk-1.png
public/assets/character/{character-id}/walk-2.png
public/assets/character/{character-id}/walk-3.png
public/assets/character/{character-id}/attack-1.png
public/assets/character/{character-id}/portrait.png
```

Example:

```text
public/assets/character/bob/idle.png
public/assets/character/bob/walk-1.png
public/assets/character/bob/walk-2.png
public/assets/character/bob/walk-3.png
public/assets/character/bob/attack-1.png
public/assets/character/bob/portrait.png
```

Each enemy follows the same minimal movement pattern:

```text
public/assets/enemy/{enemy-id}/idle.png
public/assets/enemy/{enemy-id}/walk-1.png
public/assets/enemy/{enemy-id}/walk-2.png
public/assets/enemy/{enemy-id}/walk-3.png
public/assets/enemy/{enemy-id}/attack-1.png
```

Each bullet or attack owns its own visual assets:

```text
public/assets/spell/{attack-id}/projectile.png
public/assets/spell/{attack-id}/hit-1.png
public/assets/spell/{attack-id}/hit-2.png
```

Character definitions will reference their default attack ID; attack definitions will reference the corresponding bullet and effect asset IDs. This keeps art selection data-driven without transferring gameplay authority to the client.

## Delivery order

1. Terrain variants and rocks.
2. Ranger idle/walk frames.
3. Slime Stage 1, Stage 2, and Stage 3 boss.
4. Arc Bolt, experience crystal, power crate, and entity shadow.
5. Guardian and Guardian Pulse.
6. Brute, combat effects, portraits, and upgrade icons.

Supply all frames for one entity together so the loader never mixes generated and production frames for the same animation.
