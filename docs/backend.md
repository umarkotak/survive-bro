# Go Backend

## Current slice

The backend foundation and initial room transport are implemented in `apps/backend`:

- Fiber v3 HTTP application and middleware.
- Official Fiber v3 WebSocket adapter.
- Structured `slog` startup and request logs.
- Live/readiness health and initial Prometheus text metrics.
- Idempotent named-room creation/lookup and inspection.
- One actor goroutine per room.
- Protocol-v1 WebSocket join, identity, ping/pong, input, leave, bounded writer queues, origin allowlist, and join/message limits.
- Authoritative 20 Hz movement, enemies, Ranger projectiles, combat, XP, match timer, and 10 Hz snapshots.
- Late joins into one shared battlefield, with a real two-client integration test.
- Graceful readiness, room notification, and HTTP shutdown.

Character selection, upgrades, rematch, and reconnection remain later milestones. The WebSocket returns `unsupported_message` or `reconnect_unavailable` instead of pretending those flows exist.

## Technology decision

Use:

- `github.com/gofiber/fiber/v3` for the Fiber app, HTTP routes, middleware, and WebSocket upgrade guard.
- `github.com/gofiber/contrib/v3/websocket` for the Fiber v3-compatible socket connection handler.

The Fiber core module detects upgrades but does not expose the full WebSocket connection API. Do not add `coder/websocket` alongside this stack.

Pinned module versions live in `apps/backend/go.mod`. The Go language/toolchain baseline is `1.26.5`.

## Run and verify

From the repository root:

```text
make backend-run
make backend-test
make backend-race
```

Default address: `:3701`.

## Environment

| Variable | Default | Meaning |
| --- | --- | --- |
| `HTTP_ADDRESS` | `:3701` | Listen address |
| `ALLOWED_ORIGINS` | `http://localhost:3702,http://127.0.0.1:3702` | Comma-separated exact WebSocket origins |
| `ROOM_TTL` | `10m` | Empty-lobby lifetime |
| `WS_JOIN_TIMEOUT` | `5s` | Time allowed for the first `join_room` message |
| `SHUTDOWN_TIMEOUT` | `10s` | Drain deadline |
| `HTTP_BODY_LIMIT_BYTES` | `65536` | HTTP request body ceiling |
| `WS_MESSAGE_LIMIT_BYTES` | `16384` | WebSocket message ceiling |
| `WS_CRITICAL_BUFFER` | `64` | Per-connection critical outgoing queue |

Origins are mandatory. Non-browser clients must send an allowed `Origin` header.

## HTTP surface

```text
GET  /health/live
GET  /health/ready
GET  /metrics
POST /api/v1/rooms
PUT  /api/v1/rooms/{roomName}
GET  /api/v1/rooms/{roomName}
GET  /ws/v1/rooms/{roomName}
```

`PUT` canonicalizes a valid room name and returns whether it was created. Repeating it is safe. The legacy random-room `POST` remains available; inspecting a room never exposes player names.
