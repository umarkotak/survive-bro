# MVP Architecture

## Runtime topology

The complete MVP runs as one static browser client plus one Go process. Redis and PostgreSQL are optional future systems, not runtime dependencies.

## Client boundaries

- React owns name/room entry, HUD, connection state, results, and the touch joystick overlay. Character selection and upgrades remain planned.
- Phaser owns the map, entities, camera, interpolation, local prediction, pooling, and visual effects.
- `NetworkClient` owns socket lifecycle, envelopes, heartbeat, reconnect, decoding, and subscriptions.
- `MultiplayerSession` owns room, identity, connection, match, and result state.
- `GameBridge` carries typed commands and UI events without DOM access from Phaser or Phaser-internal inspection from React.
- Create the Phaser instance once on gameplay entry and destroy it, listeners included, when leaving the room.

`GameScene` renders server snapshots and events. It predicts only the local player's movement, reconciles to authoritative snapshots, and interpolates remote entities. React and Phaser share session state through `GameBridge`; the bridge exposes the latest touch movement vector without publishing it through React HUD state.

Routes:

- `/`: room entry, gameplay, and results. Entering a room name idempotently creates or finds it before opening one socket.

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

The first join starts immediately and up to four players may join the running match. Planned lobby/selection/countdown/rematch transitions are intentionally deferred.

## Fixed simulation

Use a 50 ms fixed step and server time only. Send snapshots every second tick. Per tick:

1. Drain commands.
2. Apply latest inputs.
3. Move players and resolve player/obstacle collisions.
4. Update weapons and create attacks.
5. Update projectiles.
6. Move monsters and resolve obstacle collisions.
7. Resolve attacks and contact damage.
8. Process deaths and spawn XP.
9. Collect pickups and process level-ups.
10. Spawn monsters.
11. Check win/loss.
12. Emit a snapshot when due.

Allow no more than three catch-up ticks. Log `room_tick_overrun` with room, entity counts, and duration.

Use circles and rectangles for collision. Add a 128-unit spatial hash for projectile/monster and pickup lookups before caps exceed roughly 200–300 entities.

## Network model

Clients send normalized input intent with increasing sequence numbers immediately on change and at 20 Hz. The server zeros input after 250 ms without a refresh.

The local client predicts, stores unacknowledged inputs, accepts authoritative state plus `lastProcessedInput`, and replays pending input. Hard correction is acceptable for the first network milestone; smooth correction follows.

Render remote entities about 100 ms behind using at least two snapshots. Projectiles use reliable spawn/remove events and client visual extrapolation; do not include projectile positions in every snapshot.

The browser sets `binaryType = "arraybuffer"` and decodes frames directly with `DataView`. Go encodes and decodes with `encoding/binary`. Protocol v2 intentionally breaks JSON v1 compatibility, so client and server releases must deploy together.

## Content

The server validates `game-data/` at startup and fails fast for duplicate IDs, missing references/assets, unsupported effects, invalid finite/range values, invalid spawn points, or out-of-bounds obstacles. `/api/v1/content/manifest` exposes only public client data.

## Deployment evolution

Start in one region with one Go instance. If horizontal scaling becomes necessary, map room code to instance through Redis and keep every room entirely in one process. Add PostgreSQL only for durable accounts, history, progression, leaderboards, or analytics. Never store simulation ticks.
