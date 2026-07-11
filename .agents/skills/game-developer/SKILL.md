---
name: game-developer
description: Plan, implement, debug, review, and verify Survive Bro multiplayer game work across the React/Phaser client, authoritative Go server, WebSocket contracts, gameplay data, tests, and deployment. Use for any repository change involving gameplay, rooms, networking, simulation, content, reliability, or MVP milestones.
---

# Game Developer

Act as the core engineering orchestrator for this monorepo.

## Start

1. Read root `AGENTS.md`.
2. Invoke `$knowledge-center` and load only the relevant source-of-truth docs.
3. Inspect the named app and shared boundary before proposing changes.
4. Identify the current milestone, authoritative owner, contract impact, and acceptance gate.

For library, framework, SDK, API, CLI, or cloud details, fetch current version-specific documentation through Context7 before coding. For Phaser, use only APIs verified for the pinned Phaser 4 version.

## Route work

| Concern | Primary location | Authority |
| --- | --- | --- |
| Screens, HUD, input, rendering | `apps/game` | React/Phaser client |
| Rooms, simulation, combat, lifecycle | `apps/backend` | Go server |
| Cross-boundary messages | `contracts` | Versioned contract |
| Balance and content definitions | `game-data` | Runtime only when validated/loaded; otherwise design contract |
| Decisions and milestone gates | `docs` | Project documentation |

Do not move authority to simplify a caller. Client visuals may predict or interpolate, but the server resolves gameplay.

## Execute

1. Write a short plan for cross-app, protocol, or architectural changes.
2. Update the contract first when message shape or semantics change.
3. Implement the smallest end-to-end slice: server behavior, client consumption, and errors.
4. Keep fixed-tick simulation deterministic and room-owned.
5. Keep balance values data-driven; validate invalid content at startup.
6. Update relevant documentation in the same change.
7. Do not run tests, typechecks, builds, benchmarks, browser checks, or manual playtests unless the user explicitly requests verification.

The user performs manual game verification. Record verification as omitted by request instead of silently treating the change as verified.

Do not spawn agents or split work externally unless the user explicitly asks for delegation. Orchestrate by maintaining boundaries and sequencing.

## Review checklist

- Client sends intent only.
- Protocol version and error behavior are explicit.
- Queues, rates, sizes, timers, and cleanup are bounded.
- Phaser instances, subscriptions, and listeners cannot leak.
- Room state is mutated only by its actor loop.
- No goroutine exists per gameplay entity.
- Reconnect tokens are secure and never logged.
- MVP non-goals did not enter scope.
- When verification is explicitly requested, cover success, invalid state, disconnect, and cleanup where relevant.

## Handoff

State the completed outcome, changed files, verification performed, known gaps, and the next unblocked milestone. If `$caveman` is active, compress this to a few exact bullets.
