# WebSocket Protocol v1

This is the initial contract inventory. Add JSON Schema and golden examples before implementing each message family.

## Transport

Endpoint: `GET /ws/v1/rooms/{roomName}`. The first client message must be `join_room` within five seconds.

Room names are case-insensitive, trimmed, 1–24 characters, and contain only letters, numbers, `-`, and `_`. The server stores and returns the canonical uppercase form.

Before opening the socket, the client ensures the room exists:

```http
PUT /api/v1/rooms/{roomName}
```

```json
{
  "roomName": "FRIDAY-SQUAD",
  "status": "lobby",
  "created": true
}
```

The operation is idempotent. `created` is `false` when the room already exists. A successful ensure never joins a player; the subsequent WebSocket `join_room` does that.

Every message uses:

```json
{
  "v": 1,
  "type": "message_type",
  "requestId": "optional-client-request-id",
  "payload": {}
}
```

Reject unsupported versions, unknown message types, malformed payloads, invalid state transitions, and messages exceeding 16 KB with stable protocol errors.

## Client to server

- `join_room`: display name and optional reconnect token.
- `select_character`: enabled character ID; lobby only.
- `set_ready`: ready boolean; lobby only.
- `start_match`: host only when all connected players are selected and ready.
- `input`: increasing sequence and normalized `moveX`/`moveY`; running only.
- `select_upgrade`: one currently offered upgrade ID.
- `request_rematch`: host request from finished state.
- `ping`: heartbeat data.
- `leave_room`: explicit departure.

Initial implemented payloads:

```json
{
  "v": 1,
  "type": "join_room",
  "requestId": "join-1",
  "payload": {
    "displayName": "Umar",
    "reconnectToken": null
  }
}
```

`join_room` payload decoding is strict. Display names are trimmed and must contain 1–20 Unicode characters.

Input payload:

```json
{
  "v": 1,
  "type": "input",
  "payload": {
    "sequence": 154,
    "moveX": 0.707,
    "moveY": -0.707
  }
}
```

The server rejects non-finite/out-of-range axes and ignores stale sequence numbers. It normalizes diagonals and zeros input after 250 ms without a refresh.

## Server to client

- `joined`: player ID, reconnect token, canonical room name, and host flag.
- `room_state`: complete lobby state after every lobby change.
- `countdown_started`: authoritative countdown timing.
- `match_started`: map, static obstacles, initial entities, and timing.
- `snapshot`: authoritative dynamic players, monsters, pickups, team progression, timer, and processed input.
- `projectile_spawned`: origin, velocity, owner, weapon, and spawn tick.
- `projectile_removed`: projectile ID and `enemy_hit`, `obstacle_hit`, `range_expired`, or `match_ended`.
- `upgrade_offer`: three server-generated choices and deadline.
- `upgrade_applied`: resulting upgrade/stat state.
- `player_died`: death and spectator context.
- `match_ended`: outcome and result data.
- `pong`: heartbeat response.
- `error`: stable code, safe message, and request ID when present.
- `server_shutdown`: reason and deadline before closure.

The multiplayer slice implements `joined`, `room_state`, `match_started`, `snapshot`, `projectile_spawned`, `projectile_removed`, `match_ended`, `pong`, `error`, and `server_shutdown`. Character selection, upgrades, rematch, and reconnection remain planned.

Initial stable error codes:

- `room_not_found`
- `room_full`
- `join_required`
- `invalid_display_name`
- `invalid_payload`
- `invalid_message`
- `unsupported_version`
- `unsupported_message`
- `reconnect_unavailable`
- `upgrade_required` for the HTTP handshake guard
- `internal_error`

## Quick-play room lifecycle

- The first joined player starts a five-minute match immediately.
- Up to four players may join the named room while the match is running.
- A late joiner spawns around the map centre according to current player order.
- When the room becomes empty, its active match resets and the empty-room expiry timer starts.
- Joining the same room before it expires starts a fresh match if no players remain.
- The server remains authoritative for all player, enemy, projectile, damage, XP, timer, and result state.

This quick-play lifecycle intentionally replaces the earlier host/ready/countdown requirement for the current MVP.

## Snapshot rules

Snapshots include `tick`, `serverTimeMs`, authoritative player movement and health, monsters, pickups, team level/XP, remaining time, and each local player's `lastProcessedInput`. They do not repeat static obstacles or projectile positions.

```json
{
  "v": 1,
  "type": "snapshot",
  "payload": {
    "tick": 42,
    "serverTimeMs": 1780000000000,
    "players": [
      {
        "id": "p_abc",
        "displayName": "Umar",
        "x": 1600,
        "y": 900,
        "velocityX": 220,
        "velocityY": 0,
        "facing": "right",
        "hp": 100,
        "maxHp": 100,
        "alive": true,
        "lastProcessedInput": 154,
        "kills": 2
      }
    ],
    "monsters": [
      { "id": 7, "x": 2100, "y": 840, "hp": 20, "maxHp": 40 }
    ],
    "pickups": [
      { "id": 4, "x": 2020, "y": 810 }
    ],
    "team": {
      "level": 2,
      "experience": 3,
      "experienceRequired": 22,
      "totalKills": 4
    },
    "remainingMs": 286400
  }
}

`match_started` contains the room name, map dimensions, match start time, and static obstacle list. `projectile_spawned` contains origin, velocity, owner, and spawn tick. `projectile_removed` identifies the projectile and removal reason.

## Contract workflow

For each milestone:

1. Specify payload and error codes here or in a linked JSON Schema.
2. Add valid and invalid golden examples under `contracts/examples/`.
3. Implement Go encoding/validation.
4. Implement TypeScript decoding with exhaustive message handling.
5. Add cross-boundary integration tests.
