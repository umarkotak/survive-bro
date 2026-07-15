# MVP Architecture

## Runtime topology

The complete MVP runs as one static browser client plus one Go process. Redis and PostgreSQL are optional future systems, not runtime dependencies.

## Client boundaries

- React owns the persisted local callsign profile, room browsing/creation, HUD, connection state, results, and the touch joystick overlay. Room-browser polling is response-driven: only one room-list request may be in flight, and the next request is scheduled after the prior request settles.
- Phaser owns the map, entities, camera, interpolation, local prediction, pooling, and visual effects.
- `NetworkClient` owns socket lifecycle, envelopes, heartbeat, reconnect, decoding, and subscriptions.
- `MultiplayerSession` owns room, identity, connection, match, and result state.
- `GameBridge` carries typed commands and UI events without DOM access from Phaser or Phaser-internal inspection from React.
- Create the Phaser instance once on gameplay entry and destroy it, listeners included, when leaving the room.

`GameScene` renders server snapshots and events. It predicts only the local player's movement, reconciles to authoritative snapshots, and interpolates remote entities. Moving projectiles remain spawn/remove-event driven. Low-count lingering beams are authoritative geometry in snapshots so late joiners see active beams; the server owns segment collision and per-enemy damage intervals. React and Phaser share session state through `GameBridge`; the bridge exposes the latest touch movement vector without publishing it through React HUD state.

Routes:

- `/`: cinematic main menu, local callsign login, account summary, and a dismissible room-browser overlay. Opening the lobby does not navigate; it hides the other menu controls while active. Room polling begins only while this overlay is visible. This profile remains device-local and does not introduce a backend account system.
- `/lobby`: legacy direct-entry compatibility only; it is normalized to `/` and opens the room-browser overlay when a saved callsign exists.
- `/armory`: reserved client placeholder; persistent armory behavior remains out of scope.

## Server boundaries

The global room manager synchronizes only the room-reference map. Each room has one goroutine and command channel and exclusively mutates its lobby, connections, simulation entities, timers, and RNG.

Use `github.com/gofiber/fiber/v3` for the HTTP application and WebSocket upgrade routes. Use the official compatible `github.com/gofiber/contrib/v3/websocket` connection handler after the Fiber upgrade guard. This user-selected stack replaces the initial `coder/websocket` proposal; do not run two WebSocket stacks.

WebSocket application traffic uses the custom binary protocol v2 at `/ws/v2/rooms/{roomName}`. It uses fixed message IDs, little-endian numeric fields, UTF-8 length-prefixed strings, float32 world values, and bounded entity counts. Do not route realtime frames through JSON, base64, reflection codecs, or a browser serialization runtime. HTTP remains JSON under `/api/v1`; Fiber uses ByteDance Sonic globally for that JSON boundary.

Current quick-play room states:

```text
lobby -> running -> finished
  ^         |          |
  +---------+----------+  (fresh join after an empty/reset match)
```

The first join starts immediately and up to six players may join the running match. Each join contributes one bounded shared team life. The room actor alone reserves lives, advances solo or proximity-assisted resurrection progress, restores health, and enforces post-resurrection immunity; clients render replicated state only. Character definitions own resurrection duration, radius, and immunity duration so this lifecycle remains data-driven. Each room retains one validated level definition containing duration, terrain/obstacle asset IDs, obstacle layout, and ordered timed events. Characters, spells, and enemies are stable server-owned definitions; `spawn_rate` events independently replace rate, cap, and weighted enemy composition. `meteor_shower` hazards are simulated and damage players on the server; compact warning/linger state is replicated for Phaser rendering. Shared XP drives a team level, while all combat and movement attributes are stored and upgraded per player. Level-up and treasure rewards enter a room-owned synchronized phase: simulation elapsed time and entity updates freeze, each player receives a private three-card offer, and a wall-clock deadline continues through the pause. Planned lobby/countdown/rematch transitions remain deferred.

## Fixed simulation

Use a 50 ms fixed step and server time only. Send snapshots every second tick. Per tick:

When a reward phase is active, increment only the replication tick, resolve selections or the `50`-second deadline, and skip the normal operation order below. React blocks the world with the choice overlay and Phaser skips prediction, interpolation, and projectile extrapolation until the owning player's authoritative `upgrade_applied` arrives.

1. Drain commands.
2. Apply latest inputs.
3. Move players and resolve player/obstacle collisions.
4. Advance eligible resurrections, then update weapons and create attacks.
5. Update projectiles.
6. Move monsters and resolve obstacle collisions.
7. Resolve attacks and contact damage.
8. Process deaths and spawn XP.
9. Attract nearby XP crystals, collect pickups and power crates, and process level-ups.
10. Spawn monsters.
11. Check win/loss, treating a pending solo auto-resurrection as recoverable.
12. Emit a snapshot when due.

Allow no more than three catch-up ticks. Log `room_tick_overrun` with room, entity counts, and duration.

Checkpoint observability records total tick duration plus movement, weapon targeting, projectile movement, broad phase, narrow phase, enemy AI, pickup, and spawning durations. It also records collision candidates/checks/results, entity gauges, snapshot construction/encoding, encoded bytes, WebSocket enqueue duration, queue depth, dropped snapshots, and critical queue failures. Broad-phase duration remains zero until the spatial-hash checkpoint is implemented.

Use circles and rectangles for collision. Add a 128-unit spatial hash for projectile/monster and pickup lookups before caps exceed roughly 200–300 entities.

Monster movement uses a deterministic 128-unit local grid for soft separation. It queries only neighboring cells, accumulates bounded corrections, and resolves the corrected positions against world obstacles; it does not introduce pairwise rigid-body physics.

## Network model

Clients send normalized input intent with increasing sequence numbers immediately on change and at 20 Hz. The server zeros input after 250 ms without a refresh.

The local client predicts, stores unacknowledged inputs, accepts authoritative state plus `lastProcessedInput`, and replays pending input. Hard correction is acceptable for the first network milestone; smooth correction follows.

Render remote entities about 100 ms behind using at least two snapshots. Projectiles use reliable spawn/remove events and client visual extrapolation; do not include projectile positions in every snapshot.

Lingering beams and explosions are authoritative entities replicated as compact snapshot geometry. Characters reference reusable spell IDs rather than owning spell implementations. Players can hold multiple spell IDs, with one active default until inventory selection is implemented.

The browser sets `binaryType = "arraybuffer"` and decodes frames directly with `DataView`. Go encodes and decodes with `encoding/binary`. Protocol v2 intentionally breaks JSON v1 compatibility, so client and server releases must deploy together.

## Content

The target content model is one global glossary covering attributes, modifiers, characters, spells, buffs, enemies, levels/events, and future artifacts. Each player has five spell slots and five buff slots. Spell/buff levels resolve through the shared modifier engine; clients never calculate authoritative inventory effects.

Spell definitions are reusable by players and enemies. Enemy spell loadouts select server-owned attacks; enemy projectiles target players and use the same bounded projectile lifecycle without transferring combat authority to clients.

Authoritative enemy damage results are emitted once per tick through binary message `74 damage_applied_batch`, chunked to at most `256` results per frame. Phaser uses monster-target results for bounded, pooled floating damage text. Snapshots remain the authority for lasting HP state; damage events are presentation input only.

`game-data/game.json` is the single editable content source. The server reads it once at startup; characters, spells, enemies, and levels are decoded with Sonic and fail startup on invalid required values, references, event types, ordering, or timing. Buffs, modifiers, and inventory remain design contracts until those gameplay systems are implemented. Public endpoints expose selection-safe data only.

## Deployment evolution

Start in one region with one Go instance. If horizontal scaling becomes necessary, map room code to instance through Redis and keep every room entirely in one process. Add PostgreSQL only for durable accounts, history, progression, leaderboards, or analytics. Never store simulation ticks.
