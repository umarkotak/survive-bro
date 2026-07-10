.PHONY: backend-format backend-run backend-test backend-race backend-bench game-dev game-typecheck game-test game-build

BACKEND_DIR := $(CURDIR)/apps/backend
BACKEND_GOCACHE := $(BACKEND_DIR)/.gocache
GAME_DIR := $(CURDIR)/apps/game

backend-format:
	cd $(BACKEND_DIR) && gofmt -w cmd internal

backend-run:
	cd $(BACKEND_DIR) && GOCACHE=$(BACKEND_GOCACHE) go run ./cmd/game-server

backend-test:
	cd $(BACKEND_DIR) && GOCACHE=$(BACKEND_GOCACHE) go test ./...

backend-race:
	cd $(BACKEND_DIR) && GOCACHE=$(BACKEND_GOCACHE) go test -race ./...

backend-bench:
	cd $(BACKEND_DIR) && GOCACHE=$(BACKEND_GOCACHE) go test ./internal/protocol -run '^$$' -bench 'Benchmark(Encode|Decode)Snapshot(Binary|SonicJSON)$$' -benchmem

game-dev:
	cd $(GAME_DIR) && bun run dev

game-typecheck:
	cd $(GAME_DIR) && bun run typecheck

game-test:
	cd $(GAME_DIR) && bun run test

game-build:
	cd $(GAME_DIR) && bun run build
