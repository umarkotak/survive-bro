# Multiplayer Bullet-Hell MVP Specification

This document extracts the binding product rules from the initial design brief. Detailed delivery sequencing lives in `implementation-plan.md`; system boundaries live in `architecture.md`.

## Product

Build a browser-first, cooperative, top-down survival game for one to six anonymous players. Level 1 lasts fifteen minutes. The team wins if at least one player is alive when the level-ending event fires and loses when all players are dead.

Controls:

- WASD or arrow keys for movement.
- A virtual joystick for touch/mobile movement.
- Number keys `1`, `2`, and `3` for upgrades.
- Mouse for lobby and character selection.

## Player journey

1. Open the Heavy Armament main menu and set a local callsign if one is not already saved on the device.
2. Choose `Play` to open the dismissible lobby overlay without leaving the main-menu scene.
3. Browse joinable rooms or create a generated five-letter room after selecting a level.
4. Select a character, then join immediately; up to five more players may join the same match.
5. Survive until the level-ending event. A dead solo player resurrects automatically while a dead squad member requires a living teammate nearby; the squad loses when nobody can complete a resurrection.
6. View the shared result and return to room entry.

## Room rules

- Capacity is 1–6 players.
- Names are case-insensitive, canonicalized to uppercase, 1–24 characters, and contain only letters, numbers, `-`, and `_`.
- The first completed WebSocket join becomes host.
- The first player starts the match immediately; late joins are allowed while capacity remains.
- Late joiners spawn near the map centre in the same authoritative field.
- Character selection is data-driven. Ranger is currently available; Guardian remains planned.
- Host status has no gameplay effect in quick play.
- Expire an empty lobby after 10 minutes.
- Reset an empty active match; joining again before expiry starts a fresh match.

## Match constants

| Property | MVP value |
| --- | ---: |
| Level 1 duration | 15 minutes |
| Map | 3200 x 1800 units |
| Simulation | 20 ticks/second |
| Snapshots | 10/second |
| Rendering | Up to 60 FPS |
| Spawn centre | 1600, 900 |
| Spawn radius | 80 units |
| Team lives added per joining player | 1 (maximum 6) |
| Resurrection duration | 2 seconds |
| Resurrection radius | 120 units |
| Resurrection health | 50% max HP |
| Resurrection immunity | 5 seconds |

Friendly fire, player collision, and player-controlled pausing are disabled. The authoritative room does enter a synchronized reward pause for level-up and treasure choices. Dead players cannot act. Each newly joined player adds one shared team life, capped at six. A life is reserved when a dead player enters resurrection and consumed when resurrection completes. Solo resurrection progresses automatically; with multiple players it progresses only while a living teammate remains inside the dead player's resurrection radius, and resets when no teammate is in range. A full squad wipe still loses because no living teammate can complete a multiplayer resurrection. When a teammate is outside the local camera, the client shows an edge marker pointing toward their rough location.

## Initial content

### Ranger

- HP `100`, speed `220`, pickup radius `120`, weapon `fireball`.
- Fireball: damage `20`, cooldown `750 ms`, base speed `700`, range `700`, radius `10`, burst `1`, directions `1`.
- Burst caps at two and directions cap at four. Directions use a centered `10°` spread; burst projectiles use a small centered `3°` separation.
- Target nearest enemy; each trajectory is straight and non-homing.
- Remove on enemy hit, obstacle hit, or maximum range.

### Frieren

- HP `90`, speed `210`, pickup radius `125`, weapon `soul-track`.
- Soul Track is a server-authoritative beam: damage `18`, cooldown `1500 ms`, length `520`, width `32`, linger `1000 ms`, damage interval `500 ms`, directions `1`.
- Beam length is also its targeting range; runtime content loading derives the common weapon range from `beam_length`.
- Every enemy overlapping the beam is damaged independently on its contact interval. The beam is replicated as active geometry in snapshots; the client does not report hits.
- Deterministic spell levels improve length, cooldown, linger duration, width, directions, then damage.

### Catapult

- HP `115`, speed `195`, pickup radius `115`, default spell `rocket`.
- Rocket: impact damage `20`, explosion-tick damage `30`, cooldown `1600 ms`, speed `480`, range `850`, projectile radius `12`, explosion radius `80`, linger `1000 ms`, damage interval `500 ms`, directions `1`.
- Its small rectangular projectile applies direct impact damage to the enemy it hits, then explodes. Obstacle and range impacts still create the explosion without direct damage. Every enemy overlapping the lingering explosion takes authoritative damage on its contact interval.
- Spells are independent content. Every character has starting spell IDs and one default active spell; acquisition and switching remain a later inventory milestone.

### Guardian

- HP `140`, speed `180`, pickup radius `100`, weapon `guardian_pulse`.
- Guardian Pulse: damage `14`, cooldown `1200 ms`, radius `135`.
- Server damages all enemies in range; the client pulse is visual only.

All source sprites face right. Horizontal movement sets facing; vertical movement preserves it; attacks do not change it. Ranger idle uses one static image, walking loops `walk-1 -> walk-2 -> walk-3 -> walk-2` at `160 ms` per frame, and a Fireball spawn temporarily overrides movement with `attack-1` for `140 ms`.

### Level 1 Slimes

- Stage 1 (`0:00`): HP `60`, speed `80`, contact damage `10`, radius `24`, XP `1`.
- Stage 2 (`1:00`): joins the Stage 1 mix and becomes dominant over the run; HP `90`, speed `92`, contact damage `16`, radius `30`, XP `2`.
- Slime King base values are loaded from `game-data/game.json`: HP `2400` before event/global multipliers. Level 1 boss encounters occur at `3:00`, `7:00`, `11:00` (two kings), and `14:00`; every non-ending king guarantees a treasure chest and only the final sovereign ends the level when killed. `15:00` remains the fixed fallback.
- Stage 1 Slimes and Slime Kings use the reusable `slime-punch` melee spell. Slime Punch deals base damage `14`, has a `1200 ms` cooldown, and can hit the nearest living player only within `90` centre-to-centre units. An authoritative `monster_attacked` event triggers a brief client squash/lunge cue without moving hit timing or damage authority to Phaser. Greater Slimes retain `enemy-slime-ball`: base damage `18`, cooldown `1000 ms`, speed `360`, range `360`, and radius `12`. Both attacks respect player armor, immunity, and boss damage multipliers; the server remains authoritative.
- While boss-event monsters are alive, compact cards below the top-right menu show the boss sprite and authoritative health. Concurrent bosses share the row as equal-width columns.
- Level 1 is a 15-minute escalating run. Bosses arrive at `3:00`, `7:00`, `11:00` (two kings), and `14:00`; the final sovereign ends the level when killed. Spell chests arrive at `2:30`, `6:30`, and `10:30`. Meteor acts begin at `5:00`, `9:00`, and `12:30`. Hidden mechanical events raise spawn pressure, enemy durability, and normal treasure cadence between visible timeline beats. At `15:00`, the fixed end event remains a fallback.
- Every Slime targets the nearest living player, moves directly, slides around obstacles, and drops XP on death.
- Monsters use soft local separation: nearby enemies gently push apart while retaining substantial overlap and continuing to pursue players. Separation is not rigid-body collision and does not block the swarm.
- Enemy armor is flat damage reduction applied independently to projectile impact, normal projectile, beam, and explosion hits. All current enemies default to armor `1`; a valid hit always deals at least `1` damage.
- Every non-ending boss-event kill drops a power crate regardless of the event-driven normal cadence. Level 1 moves normal drops from every 24 kills toward every 16 kills as pressure rises; higher enemy volume still increases rewards without making synchronized pauses constant.
- Every authoritative projectile, impact, beam, or lingering-explosion hit on an enemy emits a bounded binary damage result. Clients show a small red floating damage value at that enemy; the number is cosmetic and never feeds hit detection.

### Manual test content

- `dummy-tester` is a durable selectable character using Ranger visuals and Fireball. It has `1000` HP, `75%` armor, speed `260`, regeneration `10 HP/s`, and pickup radius `160`.
- `test-boss` is a ten-minute Boss Damage Lab. One Slime King spawns at the beginning with `1000×` base health while a continuous mixed Stage 1/Stage 2 swarm runs at `1.2/sec` with a base cap of `35`. The HUD exposes a test-only **Auto level up** button that advances the team level and opens the normal synchronized reward choice.
- `spell-lab` is a ten-minute acquisition/upgrade test with nine all-pool spell chests. Each rolls three random unowned choices from Fireball, Soul Track, Rocket, Heavy Aura, Meteorite, and Tracking Beam. Owned spells auto-cast independently; Heavy Aura is always on. Full inventories convert later chests to treasure offers that may level a specifically identified owned spell.

### Meadow

- Use a seamless repeating ground texture.
- Place ten configured large rocks, each visually `220 x 180` with collision radius `65`.
- Keep the central player-spawn area clear.
- Rocks block players, monsters, and projectiles.
- Send static obstacles once in `match_started`, not every snapshot.

## Enemy spawning

- Spawn at least 600 units from every player, preferably 700–900 units from the target player.
- Try at most ten candidate points, then skip the spawn tick.
- Scale counts by `1 + 0.55 * (playerCount - 1)`; do not scale contact damage.

| Match time | Base rate | Maximum living |
| --- | ---: | ---: |
| 0:00–1:00 | 1.2/sec | 55 |
| 1:00–2:00 | 1.8/sec | 85 |
| 2:00–3:30 | 2.6/sec | 120 |
| 3:30–5:00 | 3.3/sec | 155 |
| 5:00–7:30 | 4.1/sec | 190 |
| 7:30–9:00 | 4.8/sec | 220 |
| 9:00–11:30 | 5.6/sec | 240 |
| 11:30–13:00 | 6.4/sec | 250 |
| 13:00–15:00 | 7.2/sec | 260 |

Hidden hardening events at `2:00`, `4:00`, `7:30`, `10:00`, `12:00`, and `13:30` compound to approximately `7.48×` base enemy health and `1.41×` base speed by the final act. Boss-specific multipliers compose with the active global values.

## Experience and upgrades

Experience and team level are shared. Threshold: `round(8 + 5 * level^1.45)`. Attributes and reward cards are individual. Every team level and collected power crate pauses the authoritative room and gives every current player three unique eligible choices rolled independently for that player. Number keys `1–3` and touch/click select a card. The room resumes when everyone has selected or after `50` wall-clock seconds; each unresolved player receives their own first offered card. A player joining during the phase receives a new personal offer and joins the pending count, while leaving removes that requirement. No level-up or treasure upgrade is granted immediately.

XP crystals inside the fixed `120`-unit pickup radius move toward the nearest living player at `900` units/second and collect at `32` units. Normal power-crate cadence is level-event driven; levels without a `treasure_rate` event retain the default one crate per twelve team kills.

Random effects are: max health `+20` and heal `20`; armor `+5` percentage points (cap `60%`); movement speed `+8%` base (cap `+80%`); regeneration `+1 HP/s`; attack buff `+10%`; cooldown reduction `+8` percentage points (cap `60%`); Fireball damage `+4`; projectile speed `+70`; burst `+1` (cap `2`); or directions `+1` (cap `4`). Capped upgrades are removed from the eligible roll.

Offers and validation remain server-authoritative: stale offer IDs, duplicate selections, and indexes outside the offered cards are rejected. Every applied personal upgrade emits an authoritative event identifying whether it came from a team level-up or treasure chest. The owning client shows a temporary top-centre notification and keeps an in-memory history for the current run.

### Inventory target

The accepted inventory holds a configurable maximum of four unique spells and five buffs. Ranger starts with Fireball level 1. Spell chests currently acquire unowned spells only; general level-up/treasure inventory migration remains pending. Detailed modifier ordering and the remaining runtime migration gate live in `inventory-and-modifiers.md`.

## Reliability and security

- Join within five seconds of opening the room socket.
- Ping every 10 seconds; close after 30 seconds without valid traffic.
- Preserve a disconnected in-match entity for 15 seconds, stopped and invulnerable.
- Reconnect tokens are cryptographically secure, room/player scoped, single-controller, secret, and invalidated at room close.
- Incoming message limit: 16 KB. Input rate: at most 30/second. Critical outgoing queue: at most 64.
- Drop superseded snapshots, never critical events; disconnect persistently slow clients.
- Display names are trimmed plain text, 1–20 characters.
- Use an origin allowlist and validate every enum and state transition server-side.

## Result data

Return survival time, team level, total kills, and per-player damage dealt, damage taken, and character. Do not persist results in the MVP.

## Explicit non-goals

Do not add P2P/WebRTC, accounts, OAuth, databases, Redis, matchmaking, chat, PvP, gamepads, additional maps, unlocks, cosmetics, equipment, procedural maps, voice, Kubernetes, microservices, Protobuf, replay, or client-side anti-cheat beyond server authority. The accepted realtime transport is the custom binary WebSocket v2 contract.
