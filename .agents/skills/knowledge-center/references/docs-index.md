# Documentation Index

Paths are relative to the repository root.

| Need | Read | Status meaning |
| --- | --- | --- |
| Player journey, game rules, balance baseline, limits, non-goals | `docs/mvp-spec.md` | Binding MVP specification |
| Client/server ownership, simulation, networking, scaling boundaries | `docs/architecture.md` | Accepted architecture |
| Milestones, deliverables, tests, acceptance gates | `docs/implementation-plan.md` | Planned work; not completion evidence |
| Backend setup, environment, routes, and current implementation status | `docs/backend.md` | Implemented backend operating reference |
| Game setup, offline loop, client boundaries, and current implementation status | `docs/game.md` | Implemented browser-game operating reference |
| WebSocket envelope, messages, payload evolution | `contracts/websocket-events.md` | Initial shared contract inventory |
| Implemented balance and content | `game-data/*` | Source of truth once files exist |
| Repo-wide agent workflow | `AGENTS.md` | Binding repository instructions |
| Client implementation | `apps/game` | Current browser behavior |
| Server implementation | `apps/backend` | Current authoritative behavior |

## Useful searches

```text
rg -n "RoomState|room_state|countdown|reconnect" apps contracts docs
rg -n "tick|snapshot|lastProcessedInput|interpol" apps contracts docs
rg -n "damage|cooldown|experience|upgrade|spawn" game-data apps docs
rg -n "TODO|FIXME|non-goal|future" apps contracts game-data docs
```

Do not infer that an item in the implementation plan exists. Confirm it in code, tests, configuration, or generated artifacts.
