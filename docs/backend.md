# Go Backend

## Current slice

The backend foundation and initial room transport are implemented in `apps/backend`:

- Fiber v3 HTTP application and middleware.
- Wildcard HTTP CORS (`Access-Control-Allow-Origin: *`) without credentials.
- ByteDance Sonic as Fiber's global HTTP JSON encoder/decoder.
- Official Fiber v3 WebSocket adapter.
- Structured `slog` startup and request logs.
- Live/readiness health and initial Prometheus text metrics.
- Idempotent named-room creation/lookup and inspection.
- One actor goroutine per room.
- Binary protocol-v2 WebSocket join, identity, ping/pong, input, snapshots, bounded writer queues, origin allowlist, and join/message limits.
- Authoritative 20 Hz movement, enemies, Ranger projectiles, combat, XP, match timer, and 10 Hz snapshots. Newly spawned Arc Bolts gain `70` speed for every team level above level 1.
- Late joins into one shared battlefield, with a real two-client integration test.
- Graceful readiness, room notification, and HTTP shutdown.

Character selection, upgrades, rematch, and reconnection remain later milestones. The WebSocket returns `unsupported_message` or `reconnect_unavailable` instead of pretending those flows exist.

## Technology decision

Use:

- `github.com/gofiber/fiber/v3` for the Fiber app, HTTP routes, middleware, and WebSocket upgrade guard.
- `github.com/gofiber/contrib/v3/websocket` for the Fiber v3-compatible socket connection handler.
- `github.com/bytedance/sonic` for all backend HTTP JSON encoding and decoding.

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
| `ALLOWED_ORIGINS` | `http://localhost:3702,http://127.0.0.1:3702,https://survive-bro-dev.cabocil.com` | Comma-separated exact WebSocket origins |
| `ROOM_TTL` | `10m` | Empty-lobby lifetime |
| `WS_JOIN_TIMEOUT` | `5s` | Time allowed for the first `join_room` message |
| `SHUTDOWN_TIMEOUT` | `10s` | Drain deadline |
| `HTTP_BODY_LIMIT_BYTES` | `65536` | HTTP request body ceiling |
| `WS_MESSAGE_LIMIT_BYTES` | `16384` | WebSocket message ceiling |
| `WS_CRITICAL_BUFFER` | `64` | Per-connection critical outgoing queue |

HTTP endpoints allow requests from every origin and do not allow credentialed CORS requests. WebSocket origin validation is separate: WebSocket origins remain mandatory and must match `ALLOWED_ORIGINS`; non-browser socket clients must send an allowed `Origin` header.

For Cloudflare Tunnel, publish the API hostname as an HTTP service pointing to `http://localhost:3701`; Cloudflare terminates TLS and supports WebSocket upgrades on the same route. If deployment sets `ALLOWED_ORIGINS` explicitly, it replaces the defaults and must include `https://survive-bro-dev.cabocil.com`.

## HTTP surface

```text
GET  /health/live
GET  /health/ready
GET  /metrics
POST /api/v1/rooms
PUT  /api/v1/rooms/{roomName}
GET  /api/v1/rooms/{roomName}
GET  /ws/v2/rooms/{roomName}
```

`PUT` canonicalizes a valid room name and returns whether it was created. Repeating it is safe. The legacy random-room `POST` remains available; inspecting a room never exposes player names.

## Realtime codec evidence

WebSocket v2 uses the schema in `contracts/websocket-events.md`; it never marshals JSON. On an Apple M5 Pro with Go 1.26.5, a synthetic snapshot containing 4 players, 150 monsters, and 50 pickups measured:

| Codec | Encoded bytes | Encode time | Allocations |
| --- | ---: | ---: | ---: |
| Binary v2 encode | `3262` | `3.19–3.39 µs/op` | `1` (`4096 B/op`) |
| Sonic JSON encode | `9137` | `19.18–20.02 µs/op` | `3` (`~9660 B/op`) |
| Binary v2 decode | `3262` | `2.34–2.43 µs/op` | `12` (`8160 B/op`) |
| Sonic JSON decode | `9137` | `20.12–20.66 µs/op` | `10` (`~18.5 KiB/op`) |

This specific local benchmark shows a `64.3%` payload reduction, roughly `6x` faster encoding, and roughly `8.5x` faster decoding. It is comparative codec evidence, not a server-capacity claim.
