# Browser Game

## Current slice

`apps/game` contains the first networked multiplayer slice:

- React 19.2.7, TypeScript 7.0.2, Vite 8.1.4, and Phaser exactly `4.2.1`.
- Node 24 declarations through `.node-version`, `.nvmrc`, and package engines.
- Name and room entry before play; room ensure creates or joins the canonical named room.
- One `MultiplayerSession`, WebSocket, and `GameCanvas` per gameplay entry with complete cleanup on leave.
- Dependency-free binary WebSocket v2 codec using `ArrayBuffer` and `DataView`; realtime messages never pass through JSON.
- Responsive Meadow arena at 3200 x 1800 world units rendering authoritative players, Crawlers, pickups, projectiles, and results.
- Local Ranger movement prediction/reconciliation and remote-player interpolation.
- Responsive virtual joystick movement on touch/mobile layouts; portrait phones place it at the bottom center for easier reach, while landscape keeps it bottom-left. Keyboard controls remain available.
- Camera-edge teammate badges/arrows for players outside the viewport.
- One consistent desktop/mobile HUD layer above Phaser: a 10-pixel full-width XP bar at the absolute top edge, health and level at top-left, and a top-right menu button opening a modal with the leave action.
- Levels 1–4 render 1–4 authoritative Arc Bolt trajectories, capped at four. Level and power-crate movement stats come from snapshots so local prediction uses the same resolved speed as the server.
- XP crystals interpolate along the server-owned magnet pull toward a nearby player. Gold power crates render as distinct pickups and grant bounded haste, armor, or magnet effects.
- React HUD updates through `GameBridge` at no more than 10 Hz; React does not render world entities or update every frame.
- Optional Checkpoint 1 diagnostics at `?debug=1` show smoothed FPS, visible/active gameplay sprites, active projectiles, snapshot interval, binary decode time, heartbeat RTT, and latest frame bytes. The overlay and its Phaser counters stay disabled without that query parameter.

The Go server owns movement, enemies, projectiles, damage, XP, and match outcome. The client sends normalized input intent only.

## Development art

The current scene generates compact textures in `BootScene` so the loop is playable without blocking on asset production. These are placeholders, not the final asset pack. Production images still follow the dimensions and art rules from the MVP source document.

## Run and verify

Verification is manual by project-owner direction. Do not run tests, typechecks, builds, benchmarks, or browser checks unless explicitly requested.

Use Node 24 and Bun:

```text
bun install --frozen-lockfile
make game-dev
make game-typecheck
make game-test
make game-build
```

Vite serves on `http://localhost:3702`. The browser defaults to the Cabocil development backend:

```text
VITE_API_BASE_URL=https://survive-bro-dev-api.cabocil.com
VITE_WEBSOCKET_BASE_URL=wss://survive-bro-dev-api.cabocil.com
```

Both variables are public build-time configuration, not secrets. Copy `apps/game/.env.example` to an ignored `.env.local` to override them. For a direct local backend use `http://localhost:3701` and `ws://localhost:3701`. Vite's `/api`, `/health`, `/metrics`, and `/ws` development proxies remain available for same-origin tooling.

Connection failures report the failed stage and target. HTTP failures include status and the server's safe error message; WebSocket failures include the WSS URL, frontend origin, and browser close code/reason when exposed. Browsers intentionally hide failed WebSocket handshake response bodies, so close code `1006` is accompanied by origin and Cloudflare checks.

The Vite development and preview servers explicitly allow the deployment hostname `survive-bro-dev.cabocil.com`. Localhost and IP hosts remain covered by Vite's built-in defaults; do not replace the exact hostname allowlist with `true`.

## Main boundaries

- `src/App.tsx`: room entry, React HUD, leave action, and result overlay.
- `src/components/GameCanvas.tsx`: Phaser lifetime only.
- `src/components/VirtualJoystick.tsx`: pointer-captured mobile movement overlay with release/cancel cleanup.
- `src/network/NetworkClient.ts`: room ensure, binary socket lifecycle, heartbeat, and inputs.
- `src/network/protocol.ts`: binary v2 frame encoder/decoder and bounded schema validation.
- `src/config/network.ts`: validated HTTP/WebSocket base-URL configuration.
- `src/network/MultiplayerSession.ts`: shared network/session lifetime.
- `src/bridge/GameBridge.ts`: typed session state and low-frequency Phaser-to-React HUD updates.
- `src/game/scenes/BootScene.ts`: generated development textures.
- `src/game/scenes/GameScene.ts`: authoritative world rendering, prediction, interpolation, and teammate edge markers.
- `src/game/model.ts`: pure rendering/prediction helpers, including edge-indicator placement.

## Known next work

- Replace generated development art with the required asset manifest.
- Add production-friendly Phaser chunk loading after gameplay direction stabilizes.
- Add Guardian selection, upgrade choices, reconnect, and rematch flows.
- Expand deterministic codec, prediction, interpolation, and input-state coverage as realtime features grow.
