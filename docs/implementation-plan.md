# Multiplayer Bullet-Hell MVP Implementation Plan

## Delivery strategy

Build thin, playable vertical slices in this order:

```text
foundation -> offline fun -> lobby -> authoritative movement
-> authoritative enemies -> combat/progression -> lifecycle
-> reliability -> production evidence
```

Do not optimize distributed networking before offline play is fun, and do not add persistence before a single server reliably completes matches.

## Current implementation checkpoint

The repository now has a quick-play vertical slice spanning the room, movement, enemy, Ranger combat/XP, and basic match-lifecycle milestones:

- Backend defaults to `:3701`; Vite defaults to `:3702` and proxies HTTP/WebSocket traffic.
- A player enters a named room. `PUT /api/v1/rooms/{roomName}` creates it if absent or returns the existing room, then the WebSocket join starts or joins its live match.
- One room goroutine owns a 20 Hz authoritative simulation and sends 10 Hz snapshots.
- WebSocket traffic uses binary protocol v2; backend HTTP JSON uses Sonic.
- Multiple clients render the same players, enemies, projectiles, pickups, XP, timer, and result.
- The local client predicts/reconciles movement, remote players interpolate, and off-screen teammates receive edge indicators.
- Focused Go tests include two real WebSocket clients observing authoritative movement; frontend tests cover edge placement and core model rules.
- Before resolved upgrade and pickup-kind fields were added, a 4-player/150-monster/50-pickup fixture measured `3262` binary bytes versus `9137` Sonic JSON bytes on Apple M5 Pro. Treat this as historical evidence until verification is explicitly requested and the updated frame is measured.

This checkpoint does not complete the milestones below. Guardian selection, upgrades, reconnect, rematch, richer results, data-file validation, load evidence, and the full production gates remain open.

## Hybrid simulation checkpoint roadmap

## Inventory and global-data migration checkpoint

This checkpoint precedes additional spells, buffs, or artifacts:

1. Freeze the schema and glossary in `game-data/game.json`.
2. Add strict Sonic decoding, cross-reference validation, deterministic modifier ordering, and startup failure for invalid data.
3. Replace Go content literals and mark the JSON status `runtime` in the same change.
4. Add authoritative player inventory state with five spell and five buff slots.
5. Replace direct random attribute rolls with add-item or level-item rewards.
6. Extend binary snapshots/events and add React inventory presentation.
7. Manually verify empty slots, full inventories, max levels, Fireball progression, level rewards, treasure rewards, reconnect/reset behavior, and modifier totals.

Do not add a second spell or artifact before steps 1–3 remove the duplicate runtime definitions.

| Checkpoint | Status | Manual gate |
| --- | --- | --- |
| 1. Observability baseline | Implemented; awaiting owner verification | Four clients expose server phase/entity/queue metrics and client `?debug=1` diagnostics sufficient to classify lag. |
| 2. Swept collision | Blocked by Checkpoint 1 gate | Fast projectiles cannot tunnel through monsters or rocks. |
| 3. 128-unit spatial broad phase | Pending | Candidate and narrow-phase counts fall without changing results. |
| 4. Batched projectile replication | Pending | One multishot creates one spawn frame; removals and damage batch per tick. |
| 5. Phaser rendering budget | Pending | Pools and culling keep desktop and portrait-phone play responsive. |
| 6. Data-driven weapons and Guardian | Pending | Ranger and Guardian remain server-authoritative. |
| 7. Individual choices | Pending | Offers are bounded, validated, non-pausing, and time out safely. |
| 8. Lifecycle and reconnect | Pending | Refresh, spectating, results, and rematch agree across clients. |
| 9. Performance evidence | Pending; requires explicit authorization | Record the 4-player/300-monster/2,000-projectile target without generalizing beyond measured hardware. |

Do not begin a dependent checkpoint until the owner confirms the preceding manual gate. Do not run automated checks, builds, browser verification, or load tests unless explicitly requested.

## Working rules

- Keep `apps/game` and `apps/backend` independently buildable and testable.
- Define WebSocket changes in `contracts/` before changing either app.
- Move gameplay constants into validated `game-data/`; avoid duplicated client balance values.
- Finish every milestone with its acceptance gate before starting dependent work.
- Keep future seams cheap, but do not implement future systems.

## Milestone 0 — Monorepo foundation

Deliver:

- Go module and server entry point in `apps/backend`, using Fiber v3 for HTTP and its official v3 WebSocket adapter.
- React/TypeScript/Vite app in `apps/game`; mount one Phaser 4.2.1 instance.
- Node 24 and Go 1.26 patch-version declarations.
- Root commands for format, lint, test, build, and local development.
- Dockerfiles, default two-service Compose, CI, structured `slog`, configuration, and binary protocol `v: 2`.
- `/health/live`, `/health/ready`, and metrics plumbing.

Gate:

- One root development command runs both apps.
- The page renders a Phaser canvas.
- Health checks and app builds/tests pass in CI.
- Redis and PostgreSQL are absent from the default runtime.

## Milestone 1 — Offline playable loop

Deliver in `apps/game` using shared content-shaped data:

- Meadow world, camera, Ranger movement/facing, boundaries, ten rocks, and collision.
- Crawler spawn/movement/obstacle behavior.
- Arc Bolt, damage, deaths, XP drops, timer, and win/loss.
- Object lifecycle that does not drive React state every frame.

Gate:

- Complete a five-minute match.
- Render at least 150 monsters without obvious instability.
- Player and monsters respect rocks and world bounds.
- Phaser is not recreated by ordinary React state changes.

Exit note: this is a disposable gameplay-validation simulation. Do not let offline authority leak into the later network contract.

## Milestone 2 — Rooms and lobby

Deliver:

- Named-room ensure (`PUT`), room inspection, and expiration. The legacy random-room `POST` may remain.
- Room actor/state machine and WebSocket join.
- Anonymous identity and full `room_state` broadcasts.
- Quick play starts on first join and accepts late joins up to capacity.

Gate:

- Four tabs can enter the same named battlefield.
- Fifth player and unknown room receive stable errors.
- Repeating room ensure is idempotent.

## Milestone 3 — Authoritative player movement

Deliver:

- Fixed 20 Hz room loop and stable tick order.
- Sequenced normalized inputs, stale-input stop, world/rock collision, and 10 Hz snapshots.
- Local prediction/reconciliation and 100 ms remote interpolation.
- Facing rules and slow-tick instrumentation.

Gate:

- Clients agree on positions; no client sends position.
- Diagonal movement has no speed advantage.
- Input release stops promptly.
- Movement remains usable under simulated 100–150 ms latency.

## Milestone 4 — Authoritative enemies

Deliver:

- Configured difficulty curve and player-count scaling.
- Server spawn validation, nearest-living-player targeting, movement, rock resolution, and contact damage cooldown.
- Monster snapshot rendering and 128-unit spatial indexing where needed.

Gate:

- All clients see the same monsters, health, damage, and death.
- Disconnect/death causes correct retargeting.
- No entity owns a goroutine.

## Milestone 5 — Combat, XP, and upgrades

Deliver:

- Ranger Arc Bolt spawn/remove protocol and client extrapolation.
- Guardian server-authoritative pulse plus client-only effect.
- XP orbs with server-owned magnet attraction, team level, Arc Bolt speed and 1–4 trajectory scaling, movement/armor/magnet level effects, bounded power crates, per-player offers, timeout selection, and four generic upgrades.
- Phaser pools for projectiles, enemies, pickups, and effects.

Gate:

- Both character kits work for every player.
- Clients cannot grant damage or XP or select an unoffered upgrade.
- Projectiles remain visually smooth between snapshots.
- Stack limits, private per-player card rolls, synchronized room pause, and the 50-second fallback are tested.

## Milestone 6 — Match lifecycle

Deliver:

- Five-minute timer, death, nearest-living-player spectating, win/loss, results, host rematch, lobby reset, and expiration.
- In-memory result statistics only.

Gate:

- Every terminal path produces the same result across clients.
- Rematch resets transient match state without replacing room identity.
- Empty and finished rooms expire on schedule.

## Milestone 7 — Connection reliability and security

Deliver:

- Heartbeats, connection status UI, 15-second reconnect grace, and exclusive reconnect-token ownership.
- Bounded writer queues with replaceable snapshots and non-droppable critical events.
- Message-size and input-rate limits, origin allowlist, schema/enum/state validation.
- Graceful shutdown: readiness false, stop new work, drain, notify, close.

Gate:

- Refresh reconnects during the grace period.
- Duplicate token use cannot control two sockets.
- Slow or malicious clients cannot block or crash a room.
- Logs never contain reconnect tokens.

## Milestone 8 — Production evidence

Deliver:

- Backend unit coverage for state, movement/collision, damage, XP/upgrades, spawning, lifecycle, and reconnects.
- Real WebSocket integration flow from room creation through reconnect and finish.
- Frontend unit tests for protocol, snapshot buffer/interpolation, input tracking, lobby reducer, and reconnect state.
- Playwright two-context lobby-to-canvas flow.
- Protocol fuzz tests, Go race CI, and a Go simulated load client.
- Metrics and a production Compose example.

Benchmark:

- 10 rooms, 4 players each, 150 living monsters per room, 10 minutes.
- No crash, race, unbounded memory, blocked command queue, or unexpected disconnect.
- Tick overruns below 5% on the measured hardware.

Gate: `go test -race ./...`, frontend checks, integration tests, browser flow, and benchmark all pass. Report actual hardware and results; do not generalize beyond the measurement.

## Definition of done

The MVP is done only when 1–6 players can browse/create/join a room, select either character, play one synchronized authoritative five-minute match with rocks, monsters, distinct attacks, shared XP, individual attributes, death/spectating, shared results, rematch, and 15-second reconnect—without Redis or PostgreSQL—and the verification gates above pass. The quick-play slice deliberately skips ready/countdown for faster entry.
