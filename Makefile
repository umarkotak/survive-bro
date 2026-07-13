.PHONY: backend-format backend-run backend-test backend-race backend-bench game-dev game-typecheck game-test game-build \
	local-start local-status local-stop local-logs \
	local-backend-build local-backend-start local-backend-status local-backend-stop \
	local-game-build local-game-start local-game-status local-game-stop

BACKEND_DIR := $(CURDIR)/apps/backend
BACKEND_GOCACHE := $(BACKEND_DIR)/.gocache
GAME_DIR := $(CURDIR)/apps/game

RUN_DIR := $(CURDIR)/.run
BACKEND_BINARY := $(BACKEND_DIR)/survive-bro-api
BACKEND_PID := $(RUN_DIR)/survive-bro-api.pid
BACKEND_LOG := $(RUN_DIR)/survive-bro-api.log
GAME_PID := $(RUN_DIR)/survive-bro-game.pid
GAME_LOG := $(RUN_DIR)/survive-bro-game.log

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

# Temporary background deployment (before installing system services).
# Vite preview serves the built frontend but is not a permanent production server.
local-start: local-backend-start local-game-start

local-status: local-backend-status local-game-status

local-stop: local-game-stop local-backend-stop

local-logs:
	tail -f $(BACKEND_LOG) $(GAME_LOG)

local-backend-build:
	cd $(BACKEND_DIR) && GOCACHE=$(BACKEND_GOCACHE) go build -o $(BACKEND_BINARY) ./cmd/game-server

local-backend-start: local-backend-build
	@mkdir -p $(RUN_DIR); \
	if test -f $(BACKEND_PID) && kill -0 "$$(cat $(BACKEND_PID))" 2>/dev/null; then \
		echo "backend already running (PID $$(cat $(BACKEND_PID)))"; \
		exit 1; \
	fi; \
	rm -f $(BACKEND_PID); \
	cd $(BACKEND_DIR); nohup $(BACKEND_BINARY) >> $(BACKEND_LOG) 2>&1 & \
	pid=$$!; echo $$pid > $(BACKEND_PID); \
	sleep 1; \
	if kill -0 $$pid 2>/dev/null; then \
		echo "backend started (PID $$pid, log $(BACKEND_LOG))"; \
	else \
		rm -f $(BACKEND_PID); \
		echo "backend failed to start; inspect $(BACKEND_LOG)"; \
		exit 1; \
	fi

local-backend-status:
	@if test -f $(BACKEND_PID) && kill -0 "$$(cat $(BACKEND_PID))" 2>/dev/null; then \
		echo "backend running (PID $$(cat $(BACKEND_PID)))"; \
	else \
		echo "backend stopped"; \
		rm -f $(BACKEND_PID); \
	fi

local-backend-stop:
	@if test -f $(BACKEND_PID) && kill -0 "$$(cat $(BACKEND_PID))" 2>/dev/null; then \
		pid=$$(cat $(BACKEND_PID)); kill -TERM $$pid; i=0; \
		while kill -0 $$pid 2>/dev/null && test $$i -lt 10; do sleep 1; i=$$((i + 1)); done; \
		if kill -0 $$pid 2>/dev/null; then echo "backend did not stop within 10 seconds (PID $$pid)"; exit 1; fi; \
		echo "backend stopped (PID $$pid)"; \
	else \
		echo "backend already stopped"; \
	fi; \
	rm -f $(BACKEND_PID)

local-game-build:
	cd $(GAME_DIR) && bun run build

local-game-start: local-game-build
	@mkdir -p $(RUN_DIR); \
	if test -f $(GAME_PID) && kill -0 "$$(cat $(GAME_PID))" 2>/dev/null; then \
		echo "frontend already running (PID $$(cat $(GAME_PID)))"; \
		exit 1; \
	fi; \
	rm -f $(GAME_PID); \
	cd $(GAME_DIR); nohup bun run preview -- --host 0.0.0.0 >> $(GAME_LOG) 2>&1 & \
	pid=$$!; echo $$pid > $(GAME_PID); \
	sleep 1; \
	if kill -0 $$pid 2>/dev/null; then \
		echo "frontend started (PID $$pid, log $(GAME_LOG))"; \
	else \
		rm -f $(GAME_PID); \
		echo "frontend failed to start; inspect $(GAME_LOG)"; \
		exit 1; \
	fi

local-game-status:
	@if test -f $(GAME_PID) && kill -0 "$$(cat $(GAME_PID))" 2>/dev/null; then \
		echo "frontend running (PID $$(cat $(GAME_PID)))"; \
	else \
		echo "frontend stopped"; \
		rm -f $(GAME_PID); \
	fi

local-game-stop:
	@if test -f $(GAME_PID) && kill -0 "$$(cat $(GAME_PID))" 2>/dev/null; then \
		pid=$$(cat $(GAME_PID)); kill -TERM $$pid; i=0; \
		while kill -0 $$pid 2>/dev/null && test $$i -lt 10; do sleep 1; i=$$((i + 1)); done; \
		if kill -0 $$pid 2>/dev/null; then echo "frontend did not stop within 10 seconds (PID $$pid)"; exit 1; fi; \
		echo "frontend stopped (PID $$pid)"; \
	else \
		echo "frontend already stopped"; \
	fi; \
	rm -f $(GAME_PID)
