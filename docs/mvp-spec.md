# Multiplayer Bullet-Hell MVP Specification

This document extracts the binding product rules from the initial design brief. Detailed delivery sequencing lives in `implementation-plan.md`; system boundaries live in `architecture.md`.

## Product

Build a browser-first, cooperative, top-down survival game for one to six anonymous players. Level 1 lasts six minutes. The team wins if at least one player is alive when the level-ending event fires and loses when all players are dead.

Controls:

- WASD or arrow keys for movement.
- A virtual joystick for touch/mobile movement.
- Number keys `1`, `2`, and `3` for upgrades.
- Mouse for lobby and character selection.

## Player journey

1. Set a display name, persisted on the device.
2. Browse joinable rooms or create a generated five-letter room after selecting a level.
3. Start immediately as Ranger. Up to five more players may join the same match.
4. Survive until the level-ending event or lose when every player dies.
5. View the shared result and return to room entry.

## Room rules

- Capacity is 1–6 players.
- Names are case-insensitive, canonicalized to uppercase, 1–24 characters, and contain only letters, numbers, `-`, and `_`.
- The first completed WebSocket join becomes host.
- The first player starts the match immediately; late joins are allowed while capacity remains.
- Late joiners spawn near the map centre in the same authoritative field.
- All players use Ranger in the current quick-play slice; Guardian and character selection remain planned.
- Host status has no gameplay effect in quick play.
- Expire an empty lobby after 10 minutes.
- Reset an empty active match; joining again before expiry starts a fresh match.

## Match constants

| Property | MVP value |
| --- | ---: |
| Level 1 duration | 6 minutes |
| Map | 3200 x 1800 units |
| Simulation | 20 ticks/second |
| Snapshots | 10/second |
| Rendering | Up to 60 FPS |
| Spawn centre | 1600, 900 |
| Spawn radius | 80 units |

Friendly fire, player collision, revive, and pause are disabled. Dead players cannot act. When a teammate is outside the local camera, the client shows an edge marker pointing toward their rough location.

## Initial content

### Ranger

- HP `100`, speed `220`, pickup radius `120`, weapon `fireball`.
- Fireball: damage `20`, cooldown `750 ms`, base speed `700`, range `700`, radius `10`, burst `1`, directions `1`.
- Burst caps at two and directions cap at four. Directions use a centered `10°` spread; burst projectiles use a small centered `3°` separation.
- Target nearest enemy; each trajectory is straight and non-homing.
- Remove on enemy hit, obstacle hit, or maximum range.

### Guardian

- HP `140`, speed `180`, pickup radius `100`, weapon `guardian_pulse`.
- Guardian Pulse: damage `14`, cooldown `1200 ms`, radius `135`.
- Server damages all enemies in range; the client pulse is visual only.

All source sprites face right. Horizontal movement sets facing; vertical movement preserves it; attacks do not change it. Ranger idle uses one static image, walking loops `walk-1 -> walk-2 -> walk-3 -> walk-2` at `160 ms` per frame, and a Fireball spawn temporarily overrides movement with `attack-1` for `140 ms`.

### Level 1 Slimes

- Stage 1 (`0:00`): HP `40`, speed `80`, contact damage `10`, radius `24`, XP `1`.
- Stage 2 (`1:00`): replaces normal Stage 1 spawns; HP `90`, speed `92`, contact damage `16`, radius `30`, XP `2`.
- Stage 3 boss (`5:00`): one boss spawns with HP `1800`, speed `62`, contact damage `28`, radius `54`, XP `30`.
- At `6:00`, the level ends and clients show `kills × 100 + team level × 250 + survived seconds` as the final score.
- Every Slime targets the nearest living player, moves directly, slides around obstacles, and drops XP on death.

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
| 0:00–1:00 | 1.0/sec | 60 |
| 1:00–2:30 | 1.8/sec | 110 |
| 2:30–4:00 | 2.7/sec | 170 |
| 4:00–5:00 | 3.5/sec | 240 |

## Experience and upgrades

Experience and team level are shared. Threshold: `round(8 + 5 * level^1.45)`. Attributes are individual. On every team level, each player receives one independently random eligible upgrade. A power crate gives one random eligible upgrade only to its collector. Gameplay never pauses.

XP crystals inside the fixed `120`-unit pickup radius move toward the nearest living player at `900` units/second and collect at `32` units. Every twelfth team kill drops a power crate.

Random effects are: max health `+20` and heal `20`; armor `+5` percentage points (cap `60%`); movement speed `+8%` base (cap `+80%`); regeneration `+1 HP/s`; attack buff `+10%`; cooldown reduction `+8` percentage points (cap `60%`); Fireball damage `+4`; projectile speed `+70`; burst `+1` (cap `2`); or directions `+1` (cap `4`). Capped upgrades are removed from the eligible roll.

Every applied personal upgrade emits an authoritative event identifying whether it came from a team level-up or treasure chest. The owning client shows a temporary top-centre notification and keeps an in-memory history for the current run.

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

Do not add P2P/WebRTC, accounts, OAuth, databases, Redis, matchmaking, chat, PvP, gamepads, additional maps, unlocks, cosmetics, inventory, equipment, procedural maps, bosses, revive, voice, Kubernetes, microservices, Protobuf, replay, or client-side anti-cheat beyond server authority. The accepted realtime transport is the custom binary WebSocket v2 contract.
