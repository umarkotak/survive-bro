# Game Content System

This is the central extension model for gameplay content. Implemented definitions live in `apps/backend/internal/simulation/level.go`; the server is authoritative and exposes safe selection metadata over HTTP.

## Stable content boundaries

### Character

A character owns a stable ID, display name, sprite-set ID, starting max health, armor, movement speed, regeneration, attack buff, cooldown reduction, pickup radius, and base spell ID. `GET /api/v1/characters` drives selection. `join_room.characterId` selects the server definition. Snapshots repeat `characterId` so every client chooses the correct sprite set.

### Spell

A spell owns damage, cooldown, projectile speed, range, radius, burst, and directions. A character references one base spell. Personal upgrades modify the player-owned resolved copy, never the shared definition.

### Enemy

An enemy owns a stable ID, name, sprite ID, score, XP drop, health, speed, collision radius, contact damage, and contact cooldown. Snapshots carry `typeId`; clients use that ID only for visuals.

### Level

A level owns its ID, name, duration, terrain asset IDs, obstacle asset IDs/layout, and ordered system events. `GET /api/v1/levels` drives room creation. A room locks its selected level for its lifetime.

### System event

Every event has a stable ID, timestamp, type, title, and player-facing description. The initial supported types are:

- `spawn_rate`: replaces the active normal-spawn configuration. It independently controls rate per second, maximum living enemies, and weighted enemy composition.
- `boss`: spawns one enemy type without changing the normal spawn configuration.
- `end`: resolves the match and score.

Events must be ordered and deterministic. Adding an event type requires validation, simulation handling, timeline presentation, protocol documentation, and a manual acceptance case. `match_started` sends the public event timeline once; snapshots only advance remaining time.

## Extension checklist

1. Add the server definition and stable IDs.
2. Add matching client sprite assets and sprite-set mapping.
3. Expose only selection-safe metadata through the content endpoints.
4. Update binary contracts before server/client payload changes.
5. Update `mvp-spec.md`, `levels.md`, and the asset manifest when player-visible behavior changes.
