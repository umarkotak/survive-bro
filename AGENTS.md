# Survive Bro Agent Guide

## Repository

Treat this repository as a monorepo.

- `apps/game`: browser client; React, TypeScript, Vite, and Phaser.
- `apps/backend`: authoritative Go game server and HTTP/WebSocket transport.
- `contracts`: versioned wire contracts and examples shared by both apps.
- `game-data`: server-owned gameplay configuration.
- `docs`: product, architecture, and delivery decisions.
- `.agents/skills`: reusable project workflows and knowledge routing.

Do not rename `apps/game` or `apps/backend` to match examples in older planning documents.

## Required workflow

1. Use `$knowledge-center` before planning or answering project-specific questions.
2. Use `$game-developer` for implementation, debugging, review, or milestone planning.
3. Use `$caveman` when the user requests terse or low-token communication.
4. Start cross-app or protocol work with a short plan. Name the contract boundary and acceptance checks before editing.
5. Change shared contracts before their producers and consumers.
6. Keep documentation in the same change when architecture, protocol, gameplay rules, or milestone status changes.

## Sources of truth

Resolve conflicts in this order:

1. The user's latest explicit instruction.
2. `contracts/` for messages crossing the client/server boundary.
3. `game-data/` entries whose top-level status is `runtime`, plus sections explicitly listed in `runtimeSections`, for loaded gameplay values. Other entries under `design-contract` remain target schema only.
4. `docs/mvp-spec.md` for product behavior and limits.
5. `docs/architecture.md` for system boundaries and invariants.
6. `docs/implementation-plan.md` for sequencing and acceptance gates.

Do not silently resolve a conflict that changes player-visible behavior or the wire protocol. State it and update the relevant source of truth.

## Architecture guardrails

- The Go server is authoritative for positions, enemies, combat, damage, XP, level progression, spawn timing, death, and results.
- Clients send intent, never authoritative position, health, damage, XP, or time.
- Each room owns one simulation goroutine. Never create one goroutine per entity.
- Use `github.com/gofiber/fiber/v3` for HTTP and the official `github.com/gofiber/contrib/v3/websocket` adapter for Fiber-native WebSocket connections.
- Keep the MVP in memory. Do not add Redis, PostgreSQL, accounts, matchmaking, P2P, or microservices.
- React owns screens and overlays. Phaser owns the rendered game world. Keep networking separate from both.
- Pin Phaser exactly to `4.2.1`. Use Node 24 LTS and a supported Go 1.26 patch release unless the user changes these decisions.
- Fetch current library or tool documentation through Context7 before using version-specific APIs. Never mix Phaser 3 examples into Phaser 4 code.
- Preserve binary WebSocket protocol version `v: 2` until an intentional compatibility decision changes it. HTTP remains JSON under `/api/v1`.
- Encode all WebSocket application messages with the binary contract in `contracts/`; never add JSON/base64 to the realtime path.
- Use ByteDance Sonic for backend HTTP JSON encoding and decoding.
- Prefer deterministic, testable simulation code and a stable fixed-tick operation order.

## Scope and quality

- Work on the smallest milestone that can pass end to end.
- Do not implement future architecture merely because a seam exists for it.
- Keep balance values data-driven and validated at startup.
- Treat bounded queues, input limits, reconnect-token secrecy, origin validation, and graceful shutdown as correctness requirements.
- Add tests close to the behavior. Run focused checks during development and the relevant app-wide checks before handoff.
- Do not claim capacity or latency targets without running the specified benchmark on the target environment.

## Communication

Lead with outcome, blockers, and evidence. Avoid narrating routine commands. When `$caveman` is active, be extremely terse but still report changed files, verification, assumptions, and blockers.
