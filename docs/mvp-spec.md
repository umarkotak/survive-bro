# Multiplayer Bullet-Hell MVP Specification

This document extracts the binding product rules from the initial design brief. Detailed delivery sequencing lives in `implementation-plan.md`; system boundaries live in `architecture.md`.

## Product

Build a browser-first, cooperative, top-down survival game for one to four anonymous players. A match lasts five minutes. The team wins if at least one player is alive when the timer reaches zero and loses when all players are dead.

Controls:

- WASD or arrow keys for movement.
- A virtual joystick for touch/mobile movement.
- Number keys `1`, `2`, and `3` for upgrades.
- Mouse for lobby and character selection.

## Player journey

1. Enter a display name.
2. Enter a room name. If it does not exist, create it; otherwise join its live battlefield.
3. Start immediately as Ranger. Up to three more players may join the same match.
4. Survive for five minutes or lose when every player dies.
5. View the shared result and return to room entry.

## Room rules

- Capacity is 1–4 players.
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
| Duration | 5 minutes |
| Map | 3200 x 1800 units |
| Simulation | 20 ticks/second |
| Snapshots | 10/second |
| Rendering | Up to 60 FPS |
| Spawn centre | 1600, 900 |
| Spawn radius | 80 units |

Friendly fire, player collision, revive, and pause are disabled. Dead players cannot act. When a teammate is outside the local camera, the client shows an edge marker pointing toward their rough location.

## Initial content

### Ranger

- HP `100`, speed `220`, pickup radius `120`, weapon `arc_bolt`.
- Arc Bolt: damage `20`, cooldown `750 ms`, base speed `700`, range `700`, radius `10`.
- Each team level above level 1 adds `70` projectile speed (`10%` of the base speed). The server applies the current level when it spawns a projectile.
- Target nearest enemy; fire one straight, non-homing projectile.
- Remove on enemy hit, obstacle hit, or maximum range.

### Guardian

- HP `140`, speed `180`, pickup radius `100`, weapon `guardian_pulse`.
- Guardian Pulse: damage `14`, cooldown `1200 ms`, radius `135`.
- Server damages all enemies in range; the client pulse is visual only.

All source sprites face right. Horizontal movement sets facing; vertical movement preserves it; attacks do not change it.

### Crawler

- HP `40`, speed `80`, contact damage `10`, contact cooldown `800 ms`, radius `24`, XP `1`.
- Target the nearest living player, move directly, slide around obstacles, and drop one XP orb on death.

The configuration may support a disabled Brute elite: HP `180`, speed `55`, damage `20`, cooldown `1000 ms`, radius `38`, XP `6`, available after 150 seconds.

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

Experience is team-shared. Threshold: `round(8 + 5 * level^1.45)`. Every surviving player levels together. Each level immediately increases Arc Bolt projectile speed; individual upgrade choices remain a later milestone.

The server offers three valid choices. Play continues. After eight seconds, apply the first offer if no selection arrives.

Initial generic, stack-limited effects:

- `damage_up`: +15% weapon damage, max 5.
- `cooldown_up`: -10% cooldown, max 5.
- `movement_speed_up`: +8% speed, max 5.
- `vitality_up`: +20 max HP and heal 20, max 5.

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

Do not add P2P/WebRTC, accounts, OAuth, databases, Redis, matchmaking, public room browsing, chat, PvP, gamepads, additional maps, unlocks, cosmetics, inventory, equipment, procedural maps, bosses, revive, voice, Kubernetes, microservices, Protobuf, replay, or client-side anti-cheat beyond server authority. The accepted realtime transport is the custom binary WebSocket v2 contract.
