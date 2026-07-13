# Survive Bro

Survive Bro is a multiplayer browser game with an authoritative Go backend and a React/Phaser frontend.

## Requirements

- Go `1.26.5`
- Node.js 24 LTS
- Bun
- GNU Make

Install the frontend dependencies once:

```bash
cd apps/game
bun install
cd ../..
```

## Development

Run the backend from the repository root:

```bash
make backend-run
```

The backend listens on `http://localhost:3701`.

In another terminal, run the frontend development server:

```bash
make game-dev
```

The frontend listens on `http://localhost:3702`.

## Temporary background deployment

Use this before installing permanent system services. It builds both applications, starts them in the background, and returns control to the terminal:

```bash
make local-start
```

Check both processes:

```bash
make local-status
```

Follow the backend and frontend logs:

```bash
make local-logs
```

Stop both processes gracefully:

```bash
make local-stop
```

The runtime PID and log files are stored in the ignored `.run/` directory.

Each application can also be managed independently:

```bash
make local-backend-start
make local-backend-status
make local-backend-stop

make local-game-start
make local-game-status
make local-game-stop
```

The frontend fallback uses Vite preview on port `3702`. It is suitable for this temporary deployment path, but Vite does not recommend it as a permanent production server.

## Backend launchd service

The permanent backend service uses `apps/backend/survive-bro-api.plist`. Its paths currently expect this checkout location:

```text
/Users/umar/umar/personal_projects/survive-bro
```

Update the absolute paths in the plist before installation if the server checkout is elsewhere.

Install and start the backend service on macOS:

```bash
cd apps/backend
make install-service
make status
```

Deploy a later backend revision:

```bash
cd apps/backend
make deploy
```

Other service commands:

```bash
make start
make stop
make restart
make logs
make uninstall-service
```

The backend listens on port `3701`. The launchd commands require administrator access through `sudo`.

## Build and verification

From the repository root:

```bash
make backend-test
make backend-race
make game-typecheck
make game-test
make game-build
```

More detailed operating and architecture documentation is available in [`docs/backend.md`](docs/backend.md), [`docs/game.md`](docs/game.md), and [`docs/architecture.md`](docs/architecture.md).
