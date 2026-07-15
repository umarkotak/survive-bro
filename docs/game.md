# Browser Game

## Current slice

`apps/game` contains the first networked multiplayer slice:

- React 19.2.7, TypeScript 7.0.2, Vite 8.1.4, and Phaser exactly `4.2.1`.
- Node 24 declarations through `.node-version`, `.nvmrc`, and package engines.
- A cinematic Heavy Armament main menu at `/` uses the supplied logo and image/video background. Its top-right command panel owns device-local callsign login/account state, and its sound control sits at bottom-right. After login, `Play` opens a titleless lobby modal over the same scene and temporarily hides every other menu control; the visually locked `Armory` button is disabled and cannot be activated.
- The full-height lobby overlay owns the live room browser, join actions, and a generated five-letter create-room modal of matching outer dimensions. Its compact operations heading contains the create action. Lobby, create squad, and character selection replace one another within a single overlay layer; backdrop dismissal of create or character selection returns to the lobby instead of closing the whole flow. While visible, room-list HTTP polling waits for each response, then waits two seconds before starting the next request; requests never overlap and polling exposes no loading-state label or layout transition.
- Data-driven character selection currently offers Ranger/Fireball and Frieren/Soul Track with separate sprite sets and starting stats.
- One `MultiplayerSession`, WebSocket, and `GameCanvas` per gameplay entry with complete cleanup on leave.
- Dependency-free binary WebSocket v2 codec using `ArrayBuffer` and `DataView`; realtime messages never pass through JSON.
- Responsive Meadow arena at 3200 x 1800 world units rendering authoritative players, three Slime stages, pickups, projectiles, and scored results.
- Local Ranger movement prediction/reconciliation and remote-player interpolation.
- Responsive virtual joystick movement on touch/mobile layouts; portrait phones place it at the bottom center for easier reach, while landscape keeps it bottom-left. Keyboard controls remain available.
- Camera-edge teammate badges/arrows for players outside the viewport.
- One consistent desktop/mobile HUD layer above Phaser: XP at the bottom, character health and level at top-left, a separate shared room-life panel immediately beside it, room menu at top-right, and an authoritative system-event timeline across the top. Phaser renders the server-owned resurrection radius and progress ring on eligible dead players. Its fixed centre arrow represents now; the line and event markers move right-to-left as time advances, and clicking a marker opens the unified event-detail modal.
- Team level-ups and treasure chests both open a non-dismissible synchronized reward overlay. Every player receives three independently randomized personal cards, selects with touch/click or keys `1–3`, and waits for the rest of the squad. Phaser freezes world prediction and effects until all players choose or the server resolves the `50`-second timeout.
- The character-stat view formats each upgradable value as `base (+current bonus) final`. Authoritative level/chest events drive a top-centre toast and a current-run history modal; this history is client-local and clears when leaving the room.
- A dependency-free Web Audio layer generates short, low-volume Fireball, player-damage, level-up, treasure, menu-hover, and menu-click sounds. Sound intent defaults on; the first pointer or keyboard interaction automatically unlocks audio, after which the menu video plays embedded audio at configured volume. The toggle uses only `Sound on/off` states and mutes both video and menu cues. Gameplay remains functional when audio is unavailable.
- Authoritative `damage_applied_batch` events drive a bounded pool of small red enemy damage numbers that rise and fade for `700 ms`; snapshots remain the lasting HP source.
- Every Slime currently casts the reusable short-range `enemy-slime-ball` spell. Phaser generates its red glob texture and brief player-hit flash locally while the server owns targeting, cooldown, range, damage, collision, and removal.
- Manual selectors include `dummy-tester` and the ten-minute `test-boss` Boss Damage Lab. The character reuses Ranger art; the level spawns a `1000×`-health Slime King plus a capped Stage 1/Stage 2 test swarm as soon as the first player enters. Its test-only **Auto level up** HUD button opens the same authoritative multiplayer choice flow as earned XP.
- XP crystals interpolate along the server-owned magnet pull toward a nearby player. Fireball burst and direction upgrades combine into the authoritative volley.
- React HUD updates through `GameBridge` at no more than 10 Hz; React does not render world entities or update every frame.
- Optional Checkpoint 1 diagnostics at `?debug=1` show smoothed FPS, visible/active gameplay sprites, active projectiles, snapshot interval, binary decode time, heartbeat RTT, and latest frame bytes. The overlay and its Phaser counters stay disabled without that query parameter.

The Go server owns movement, enemies, projectiles, damage, XP, and match outcome. The client sends normalized input intent only.

Visual work follows [ui-design-direction.md](ui-design-direction.md). Main-menu image/video selection is centralized in `src/config/menuMedia.ts`; the still image remains visible until an enabled video can play.

## Development art

`BootScene` loads production art through the centralized helpers in `src/config/assets.ts`. Files are grouped by category and content ID under `public/assets` while Phaser texture keys and server content IDs remain stable. Temporary Fireball, Rocket, explosion, pickup, crate, and shadow visuals remain generated until dedicated spell/effect assets are supplied.

Boss-event instances are marked in binary snapshots. React renders compact avatar/HP cards below the menu in an equal-column row; Phaser continues to own their world sprites.

The 256 x 256 Ranger sources render at 132 x 132 so their transparent-bound body aligns with the authoritative 30-unit player collision radius. The 256 x 256 rock sources render at 180 x 180 so their visible mass aligns with the authoritative 65-unit rock radius; gameplay hitboxes remain server-owned.

The complete file list, canvas sizes, minimal animation rules, delivery order, and future naming contract live in [`art-assets.md`](art-assets.md).

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
- Add Guardian selection, reconnect, and rematch flows.
- Expand deterministic codec, prediction, interpolation, and input-state coverage as realtime features grow.
