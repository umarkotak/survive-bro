# Binary WebSocket Protocol v2

Protocol v2 replaces every JSON WebSocket message with a schema-driven binary frame. HTTP endpoints remain JSON under `/api/v1` and are encoded/decoded with ByteDance Sonic.

This is an intentional breaking change. The binary socket endpoint is:

```text
GET /ws/v2/rooms/{roomName}
```

The removed JSON v1 socket must not silently accept v2 clients. Frontend and backend deploy together.

## Transport

Before opening the socket, the client idempotently ensures the canonical room through JSON HTTP:

```http
PUT /api/v1/rooms/{roomName}
```

Every WebSocket application message uses binary opcode `0x2`. Text frames are rejected. The first message must be `join_room` within five seconds. The maximum decoded frame remains 16 KiB.

### Frame header

All numeric fields are little-endian. Strings are UTF-8.

| Offset | Type | Meaning |
| ---: | --- | --- |
| `0` | `u8` | Protocol version, always `2` |
| `1` | `u8` | Message type ID |
| `2` | `u8` | Request-ID byte length `N` |
| `3` | `N bytes` | Optional UTF-8 request ID |
| `3 + N` | message-specific | Payload; must consume the rest of the frame |

Every payload string uses `u16 byteLength` followed by UTF-8 bytes. Collection counts use `u8` for players and `u16` for potentially numerous world entities. Decoders reject truncation, invalid UTF-8, unknown enum values, count/length overflow, non-finite floats, trailing bytes, unsupported versions, and unknown type IDs.

### Message IDs

| ID | Direction | Name |
| ---: | --- | --- |
| `1` | C→S | `join_room` |
| `2` | C→S | `leave_room` |
| `3` | C→S | `ping` |
| `4` | C→S | `input` |
| `64` | S→C | `joined` |
| `65` | S→C | `room_state` |
| `66` | S→C | `match_started` |
| `67` | S→C | `snapshot` |
| `68` | S→C | `projectile_spawned` |
| `69` | S→C | `projectile_removed` |
| `70` | S→C | `match_ended` |
| `71` | S→C | `pong` |
| `126` | S→C | `error` |
| `127` | S→C | `server_shutdown` |

## Client payloads

- `join_room`: `displayName string`, `hasReconnectToken u8`, then `reconnectToken string` only when present.
- `leave_room`: empty.
- `ping`: empty; request ID is echoed by `pong`.
- `input`: `sequence u32`, `moveX f32`, `moveY f32`.

Display names are trimmed and contain 1–20 Unicode characters. Input sequence is increasing. Movement axes must be finite and within `[-1, 1]`; the server normalizes diagonals and stops stale input after 250 ms.

## Server payloads

- `joined`: `playerId string`, `reconnectToken string`, `roomName string`, `host u8`.
- `room_state`: `status u8`, `hostPlayerId string`, `playerCount u8`, then players. Each player is `id string`, `displayName string`, `characterId string`, `flags u8` (`bit0 ready`, `bit1 connected`). Status enum: `0 lobby`, `1 running`, `2 finished`.
- `match_started`: `roomName string`, `mapId string`, `mapWidth f32`, `mapHeight f32`, `startedAtMs i64`, `obstacleCount u16`, then obstacles. Each obstacle is `id string`, `type string`, `x f32`, `y f32`, `radius f32`.
- `snapshot`: header/team fields and entity arrays described below.
- `projectile_spawned`: `projectileId u32`, `ownerId string`, `weaponId string`, `x f32`, `y f32`, `velocityX f32`, `velocityY f32`, `spawnTick u32`.
- `projectile_removed`: `projectileId u32`, `reason u8`. Reason enum: `0 enemy_hit`, `1 obstacle_hit`, `2 range_expired`, `3 match_ended`.
- `match_ended`: `outcome u8` (`0 lost`, `1 won`), `survivalMs u32`, `teamLevel u16`, `totalKills u32`.
- `pong`: empty.
- `error`: `code string`, `message string`.
- `server_shutdown`: `reason string`.

### Snapshot payload

```text
tick u32
serverTimeMs i64
playerCount u8
  repeated player:
    id string
    displayName string
    x f32, y f32
    velocityX f32, velocityY f32
    flags u8 (bit0 facing-left, bit1 alive)
    hp u16, maxHp u16
    lastProcessedInput u32
    kills u32
monsterCount u16
  repeated monster:
    id u32
    x f32, y f32
    hp u16, maxHp u16
pickupCount u16
  repeated pickup:
    id u32
    x f32, y f32
teamLevel u16
teamExperience u16
teamExperienceRequired u16
teamTotalKills u32
remainingMs u32
```

Static obstacles are sent once in `match_started`. Projectile positions are extrapolated from reliable spawn/remove events rather than repeated in snapshots.

## Quick-play lifecycle

- The first joined player starts a five-minute match immediately.
- Up to four players may join the named room while it is running.
- Late joiners spawn around map centre according to current player order.
- When the room becomes empty, its match resets and the empty-room expiry timer starts.
- The server remains authoritative for players, enemies, projectiles, damage, XP, timing, and results.

## Codec requirements

1. Go and TypeScript maintain matching golden frames for every implemented message family.
2. Hot paths reuse buffers where practical and never convert binary frames through JSON or base64.
3. The browser sets `WebSocket.binaryType = "arraybuffer"` before receiving messages.
4. Snapshot size and codec benchmarks are recorded as measured evidence, not universal capacity claims.
5. Protocol changes update this document before producers and consumers.
