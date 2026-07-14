# Binary WebSocket Protocol v2

Protocol v2 replaces every JSON WebSocket message with a schema-driven binary frame. HTTP endpoints remain JSON under `/api/v1` and are encoded/decoded with ByteDance Sonic.

This is an intentional breaking change. The binary socket endpoint is:

```text
GET /ws/v2/rooms/{roomName}
```

The removed JSON v1 socket must not silently accept v2 clients. Frontend and backend deploy together.

## Transport

The menu lists rooms through `GET /api/v1/rooms`, returning `{ "rooms": [{ "roomName", "status", "playerCount", "maxPlayers", "joinable" }] }`. Before opening the socket, the client idempotently ensures the selected canonical room:

```http
PUT /api/v1/rooms/{roomName}
```

When creating a room, the optional Sonic-JSON body is `{ "levelId": "level-1" }`. Existing rooms retain their original level. Room-list and inspection responses include `levelId`.

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
| `5` | C→S | `select_upgrade` |
| `7` | C→S | `debug_level_up` |
| `64` | S→C | `joined` |
| `65` | S→C | `room_state` |
| `66` | S→C | `match_started` |
| `67` | S→C | `snapshot` |
| `68` | S→C | `projectile_spawned` |
| `69` | S→C | `projectile_removed` |
| `70` | S→C | `match_ended` |
| `71` | S→C | `pong` |
| `74` | S→C | `damage_applied_batch` |
| `75` | S→C | `upgrade_offered` |
| `76` | S→C | `upgrade_applied` |
| `126` | S→C | `error` |
| `127` | S→C | `server_shutdown` |

## Client payloads

- `join_room`: `displayName string`, `characterId string`, `hasReconnectToken u8`, then `reconnectToken string` only when present.
- `leave_room`: empty.
- `ping`: empty; request ID is echoed by `pong`.
- `input`: `sequence u32`, `moveX f32`, `moveY f32`.
- `select_upgrade`: `offerId u32`, `choiceIndex u8` (`0–2`). The server rejects stale offer IDs, duplicate selections, indexes outside the player's authoritative offer, and selection while no upgrade phase is active.
- `debug_level_up`: empty. This development-only intent is accepted exclusively in a running `test-boss` room when no upgrade phase is active; it advances the shared team level and opens the normal synchronized upgrade flow.

Display names are trimmed and contain 1–20 Unicode characters. Input sequence is increasing. Movement axes must be finite and within `[-1, 1]`; the server normalizes diagonals and stops stale input after 250 ms.

## Server payloads

- `joined`: `playerId string`, `reconnectToken string`, `roomName string`, `host u8`.
- `room_state`: `status u8`, `hostPlayerId string`, `playerCount u8`, then players. Each player is `id string`, `displayName string`, `characterId string`, `flags u8` (`bit0 ready`, `bit1 connected`). Status enum: `0 lobby`, `1 running`, `2 finished`.
- `match_started`: room/map fields and obstacles, followed by `durationMs u32`, `eventCount u8`, then public timeline events. Each event is `id string`, `type string`, `title string`, `description string`, `atMs u32`. Public event types are `spawn_rate`, `monster_buff`, `meteor_shower`, `boss`, and `end`.
- `snapshot`: header/team fields and entity arrays described below.
- `projectile_spawned`: `projectileId u32`, `ownerId string`, `weaponId string`, `x f32`, `y f32`, `velocityX f32`, `velocityY f32`, `spawnTick u32`.
- Enemy-owned projectiles use `ownerId = "enemy:<monsterId>"`; clients render them normally and do not decide their hits.
- `projectile_removed`: `projectileId u32`, `reason u8`. Reason enum: `0 enemy_hit`, `1 obstacle_hit`, `2 range_expired`, `3 match_ended`, `4 player_hit`.
- `damage_applied_batch`: `count u16` (maximum `256`), then authoritative results. Each result is `attackerId string`, `targetType u8` (`0 monster`, `1 player`), `targetId string`, `amount u32`, `remainingHp u32`, `flags u8` (`bit0 critical`, `bit1 death`). Current runtime emits monster targets for projectile, impact, beam, and lingering-explosion damage. Clients may show cosmetic damage numbers but never infer or report hits from them.
- `upgrade_offered`: `offerId u32`, `source u8` (`0 level_up`, `1 treasure_chest`), `teamLevel u16`, `deadlineMs i64`, `pendingCount u8`, `totalCount u8`, `flags u8` (`bit0 this player selected`), `choiceCount u8` (exactly `3`), then choices. Each choice is `attribute string`, `currentValue f32`, `addedValue f32`, `finalValue f32`. This message is private to its owning player and is resent when selection progress changes.
- `match_ended`: `outcome u8` (`0 lost`, `1 won`), `survivalMs u32`, `teamLevel u16`, `totalKills u32`, `score u32`.
- `pong`: empty.
- `upgrade_applied`: `playerId string`, `source u8` (`0 level_up`, `1 treasure_chest`), `attribute string`, `baseValue f32`, `addedValue f32`, `finalValue f32`. Attribute IDs include player stats plus spell-specific projectile, beam, and explosion properties documented in `game-data/game.json`.
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
    characterId string
    x f32, y f32
    velocityX f32, velocityY f32
    movementSpeed f32
    armorPercent f32
    healthRegeneration f32
    attackBuffPercent f32
    cooldownPercent f32
    flags u8 (bit0 facing-left, bit1 alive)
    hp u32, maxHp u32
    spellDamage u16
    projectileSpeed f32
    spellBurst u8
    spellDirections u8
    lastProcessedInput u32
    kills u32
monsterCount u16
  repeated monster:
    id u32
    x f32, y f32
    typeId string
    hp u32, maxHp u32
    flags u8 (bit 0 boss-event instance)
beamCount u16
  repeated beam:
    id u32
    ownerId string
    spellId string
    x f32, y f32, angle f32, length f32, width f32
    remainingMs u32
explosionCount u16
  repeated explosion:
    id u32
    ownerId string
    spellId string
    x f32, y f32, radius f32
    remainingMs u32
meteorCount u16
  repeated meteor:
    id u32
    x f32, y f32, radius f32
    impactInMs u32, remainingMs u32
pickupCount u16
  repeated pickup:
    id u32
    kind u8 (0 experience, 1 power_crate)
    x f32, y f32
teamLevel u16
teamExperience u16
teamExperienceRequired u16
teamTotalKills u32
remainingMs u32
```

XP and team level are shared, but attributes are individual. Each team level and collected power crate pauses authoritative simulation and gives every current player three independently generated eligible choices. The cards are rolled separately per player, so teammates need not see the same attributes. The phase resolves after everyone selects or after `50` seconds; unresolved players receive the first offered choice. No level-up or treasure upgrade is granted outside this selection phase. Upgrades cover max health, armor, movement speed, regeneration, attack buff, cooldown, spell damage, projectile or beam properties, burst (maximum two), and directions (maximum four).

Static obstacles are sent once in `match_started`. Projectile positions are extrapolated from reliable spawn/remove events rather than repeated in snapshots.

## Quick-play lifecycle

- The first joined player starts the selected level immediately. Level 1 ends when its designated ending boss dies or at the six-minute fallback event.
- Up to six players may join the named room while it is running.
- Late joiners spawn around map centre according to current player order.
- When the room becomes empty, its match resets and the empty-room expiry timer starts.
- The server remains authoritative for players, enemies, projectiles, damage, XP, timing, and results.

## Codec requirements

1. Go and TypeScript maintain matching golden frames for every implemented message family.
2. Hot paths reuse buffers where practical and never convert binary frames through JSON or base64.
3. The browser sets `WebSocket.binaryType = "arraybuffer"` before receiving messages.
4. Snapshot size and codec benchmarks are recorded as measured evidence, not universal capacity claims.
5. Protocol changes update this document before producers and consumers.
