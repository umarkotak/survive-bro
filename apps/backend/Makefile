.PHONY: run bin bin-run nohup-run stopd statusd logs install-service uninstall-service start stop restart deploy status

SERVICE_NAME := survive-bro-api
PLIST := $(SERVICE_NAME).plist
SYSTEM_PLIST := /Library/LaunchDaemons/$(PLIST)
SERVICE_TARGET := system/$(SERVICE_NAME)
BINARY := $(SERVICE_NAME)

run:
	go run ./cmd/game-server

bin:
	go build -o $(BINARY) ./cmd/game-server

bin-run: bin
	./$(BINARY)

nohup-run: bin
	nohup ./$(BINARY) > $(SERVICE_NAME).log 2> $(SERVICE_NAME).error.log &

stopd:
	pkill -x $(SERVICE_NAME)

statusd:
	pgrep -af $(SERVICE_NAME)

logs:
	tail -f $(SERVICE_NAME).error.log

install-service: bin
	sudo cp $(PLIST) $(SYSTEM_PLIST)
	sudo chown root:wheel $(SYSTEM_PLIST)
	sudo chmod 644 $(SYSTEM_PLIST)
	sudo launchctl bootstrap system $(SYSTEM_PLIST)

uninstall-service:
	-sudo launchctl bootout $(SERVICE_TARGET)
	sudo rm -f $(SYSTEM_PLIST)

start:
	sudo launchctl kickstart $(SERVICE_TARGET)

stop:
	sudo launchctl kill SIGTERM $(SERVICE_TARGET)

restart:
	sudo launchctl kickstart -k $(SERVICE_TARGET)

deploy:
	git pull --rebase origin main
	go mod download
	$(MAKE) bin
	$(MAKE) restart

status:
	sudo launchctl print $(SERVICE_TARGET)
	lsof -nP -iTCP:3701 -sTCP:LISTEN
