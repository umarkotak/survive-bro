# Browser Game

## Current slice

`apps/game` contains the first networked multiplayer slice:

- React 19.2.7, TypeScript 7.0.2, Vite 8.1.4, and Phaser exactly `4.2.1`.
- Node 24 declarations through `.node-version`, `.nvmrc`, and package engines.
- Name and room entry before play; room ensure creates or joins the canonical named room.
- One `MultiplayerSession`, WebSocket, and `GameCanvas` per gameplay entry with complete cleanup on leave.
- Responsive Meadow arena at 3200 x 1800 world units rendering authoritative players, Crawlers, pickups, projectiles, and results.
- Local Ranger movement prediction/reconciliation and remote-player interpolation.
- Camera-edge teammate badges/arrows for players outside the viewport.
- Shared health, XP, team level, kills, nearby enemies, timer, and terminal result UI.
- React HUD updates through `GameBridge` at no more than 10 Hz; React does not render world entities or update every frame.

The Go server owns movement, enemies, projectiles, damage, XP, and match outcome. The client sends normalized input intent only.

## Development art

The current scene generates compact textures in `BootScene` so the loop is playable without blocking on asset production. These are placeholders, not the final asset pack. Production images still follow the dimensions and art rules from the MVP source document.

## Run and verify

Use Node 24 and Bun:

```text
bun install --frozen-lockfile
make game-dev
make game-typecheck
make game-test
make game-build
```

Vite serves on `http://localhost:3702` and proxies `/api`, `/health`, `/metrics`, and `/ws` to the Go backend on `http://localhost:3701`. The WebSocket proxy preserves the browser origin expected by the backend allowlist.

## Main boundaries

- `src/App.tsx`: room entry, React HUD, leave action, and result overlay.
- `src/components/GameCanvas.tsx`: Phaser lifetime only.
- `src/network/NetworkClient.ts`: room ensure, socket lifecycle, protocol decoding, heartbeat, and inputs.
- `src/network/MultiplayerSession.ts`: shared network/session lifetime.
- `src/bridge/GameBridge.ts`: typed session state and low-frequency Phaser-to-React HUD updates.
- `src/game/scenes/BootScene.ts`: generated development textures.
- `src/game/scenes/GameScene.ts`: authoritative world rendering, prediction, interpolation, and teammate edge markers.
- `src/game/model.ts`: pure rendering/prediction helpers, including edge-indicator placement.

## Known next work

- Replace generated development art with the required asset manifest.
- Add production-friendly Phaser chunk loading after gameplay direction stabilizes.
- Add Guardian selection, upgrade choices, reconnect, and rematch flows.
- Add broader browser automation for movement/indicator screenshots and full match completion.
