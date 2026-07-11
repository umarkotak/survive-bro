# Level Framework

Level definitions are server-owned in `apps/backend/internal/simulation/level.go`. A definition contains:

- Stable ID and display name.
- Match duration.
- Terrain and obstacle asset IDs.
- Authoritative obstacle layout.
- Enemy definitions with HP, speed, collision radius, contact damage/cooldown, and XP.
- Initial normal enemy type.
- Ordered timed events.

Supported event types are deliberately small:

- `enemy_stage`: changes the enemy used by subsequent normal spawns.
- `boss`: spawns one configured boss enemy.
- `end`: resolves the match and score.

The create-room screen loads options from `GET /api/v1/levels` and sends the selected `levelId`. Rooms retain that definition for their lifetime, late joiners enter the same timeline, `match_started.mapId` carries the selected level ID, and snapshots carry each monster `typeId` so the client can select the correct visual.

## Level 1: Slime Meadow

| Time | Event |
| ---: | --- |
| `0:00` | Spawn Slime Stage 1. |
| `1:00` | Normal spawns switch to stronger Slime Stage 2. Existing Stage 1 enemies remain. |
| `5:00` | Spawn one Slime Stage 3 boss. |
| `6:00` | End the match and show score. |

Level 1 uses the three existing terrain variants and three large-rock variants. Add future levels by defining their content and adding them to `AvailableLevels`; the room selector and public room metadata use stable level IDs.
