package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	clientws "github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v3"

	"survive-bro/apps/backend/internal/config"
	"survive-bro/apps/backend/internal/protocol"
	"survive-bro/apps/backend/internal/room"
)

func testConfig() config.Config {
	return config.Config{
		HTTPAddress:             ":0",
		AllowedOrigins:          []string{"http://localhost:3702", "http://127.0.0.1:3702"},
		RoomTTL:                 time.Minute,
		JoinTimeout:             250 * time.Millisecond,
		ShutdownTimeout:         time.Second,
		HTTPBodyLimitBytes:      64 * 1024,
		WebSocketMessageBytes:   16 * 1024,
		WebSocketCriticalBuffer: 64,
	}
}

func testRouter(t *testing.T, ready ReadyFunc) (*fiber.App, *room.Manager) {
	t.Helper()
	manager := room.NewManager(time.Minute)
	app := NewRouter(testConfig(), manager, slog.New(slog.NewTextHandler(io.Discard, nil)), ready)
	t.Cleanup(func() {
		_ = manager.Close(context.Background())
		_ = app.Shutdown()
	})
	return app, manager
}

func TestHealthEndpoints(t *testing.T) {
	app, _ := testRouter(t, func() bool { return true })

	for _, path := range []string{"/health/live", "/health/ready"} {
		request := httptest.NewRequest(http.MethodGet, path, http.NoBody)
		response, err := app.Test(request)
		if err != nil {
			t.Fatalf("GET %s error = %v", path, err)
		}
		if response.StatusCode != http.StatusOK {
			t.Fatalf("GET %s status = %d", path, response.StatusCode)
		}
	}
}

func TestReadinessReportsDraining(t *testing.T) {
	app, _ := testRouter(t, func() bool { return false })
	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/health/ready", http.NoBody))
	if err != nil {
		t.Fatalf("GET readiness error = %v", err)
	}
	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", response.StatusCode)
	}
}

func TestCreateAndInspectRoom(t *testing.T) {
	app, _ := testRouter(t, func() bool { return true })
	createResponse, err := app.Test(httptest.NewRequest(http.MethodPost, "/api/v1/rooms", http.NoBody))
	if err != nil {
		t.Fatalf("POST room error = %v", err)
	}
	if createResponse.StatusCode != http.StatusCreated {
		t.Fatalf("POST room status = %d", createResponse.StatusCode)
	}
	var created struct {
		RoomName string `json:"roomName"`
	}
	if err := json.NewDecoder(createResponse.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	inspectResponse, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/rooms/"+strings.ToLower(created.RoomName), http.NoBody))
	if err != nil {
		t.Fatalf("GET room error = %v", err)
	}
	if inspectResponse.StatusCode != http.StatusOK {
		t.Fatalf("GET room status = %d", inspectResponse.StatusCode)
	}
	var inspected struct {
		RoomName    string `json:"roomName"`
		Status      string `json:"status"`
		PlayerCount int    `json:"playerCount"`
		MaxPlayers  int    `json:"maxPlayers"`
		Joinable    bool   `json:"joinable"`
	}
	if err := json.NewDecoder(inspectResponse.Body).Decode(&inspected); err != nil {
		t.Fatalf("decode inspect response: %v", err)
	}
	if inspected.RoomName != created.RoomName || inspected.Status != "lobby" || !inspected.Joinable || inspected.MaxPlayers != 4 {
		t.Fatalf("unexpected inspect response: %#v", inspected)
	}
}

func TestEnsureNamedRoomIsIdempotent(t *testing.T) {
	app, _ := testRouter(t, func() bool { return true })
	for index, name := range []string{"friday-squad", "FRIDAY-SQUAD"} {
		response, err := app.Test(httptest.NewRequest(http.MethodPut, "/api/v1/rooms/"+name, http.NoBody))
		if err != nil {
			t.Fatalf("PUT room error = %v", err)
		}
		if response.StatusCode != http.StatusOK {
			t.Fatalf("PUT room status = %d", response.StatusCode)
		}
		var ensured struct {
			RoomName string `json:"roomName"`
			Created  bool   `json:"created"`
		}
		if err := json.NewDecoder(response.Body).Decode(&ensured); err != nil {
			t.Fatalf("decode ensure response: %v", err)
		}
		if ensured.RoomName != "FRIDAY-SQUAD" || ensured.Created != (index == 0) {
			t.Fatalf("unexpected ensure response: %#v", ensured)
		}
	}
}

func TestWebSocketJoinAndPing(t *testing.T) {
	cfg := testConfig()
	manager := room.NewManager(cfg.RoomTTL)
	created, err := manager.Create()
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	app := NewRouter(cfg, manager, slog.New(slog.NewTextHandler(io.Discard, nil)), func() bool { return true })
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- app.Listener(listener, fiber.ListenConfig{DisableStartupMessage: true})
	}()
	t.Cleanup(func() {
		_ = manager.Close(context.Background())
		_ = app.Shutdown()
		<-serverDone
	})

	header := http.Header{"Origin": []string{"http://localhost:3702"}}
	connection, response, err := clientws.DefaultDialer.Dial("ws://"+listener.Addr().String()+"/ws/v1/rooms/"+created.Code(), header)
	if err != nil {
		if response != nil {
			t.Fatalf("dial status=%d error=%v", response.StatusCode, err)
		}
		t.Fatalf("dial: %v", err)
	}
	defer connection.Close()

	join, err := protocol.NewEnvelope(protocol.TypeJoinRoom, "join-1", protocol.JoinRoomPayload{DisplayName: "Umar"})
	if err != nil {
		t.Fatalf("join envelope: %v", err)
	}
	if err := connection.WriteJSON(join); err != nil {
		t.Fatalf("write join: %v", err)
	}

	var joined protocol.Envelope
	if err := connection.ReadJSON(&joined); err != nil {
		t.Fatalf("read joined: %v", err)
	}
	if joined.Type != protocol.TypeJoined {
		t.Fatalf("first message type = %q", joined.Type)
	}
	var joinedPayload protocol.JoinedPayload
	if err := joined.DecodePayload(&joinedPayload); err != nil {
		t.Fatalf("decode joined: %v", err)
	}
	if !joinedPayload.Host || joinedPayload.RoomName != created.Code() || joinedPayload.ReconnectToken == "" {
		t.Fatalf("unexpected joined payload: %#v", joinedPayload)
	}

	seen := map[string]bool{}
	for !(seen[protocol.TypeMatchStarted] && seen[protocol.TypeRoomState] && seen[protocol.TypeSnapshot]) {
		var message protocol.Envelope
		if err := connection.ReadJSON(&message); err != nil {
			t.Fatalf("read initial multiplayer state: %v", err)
		}
		seen[message.Type] = true
	}

	ping, err := protocol.NewEnvelope(protocol.TypePing, "ping-1", struct{}{})
	if err != nil {
		t.Fatalf("ping envelope: %v", err)
	}
	if err := connection.WriteJSON(ping); err != nil {
		t.Fatalf("write ping: %v", err)
	}
	for {
		var pong protocol.Envelope
		if err := connection.ReadJSON(&pong); err != nil {
			t.Fatalf("read pong: %v", err)
		}
		if pong.Type == protocol.TypePong {
			if pong.RequestID != "ping-1" {
				t.Fatalf("unexpected pong: %#v", pong)
			}
			break
		}
	}

	leave, _ := protocol.NewEnvelope(protocol.TypeLeaveRoom, "leave-1", struct{}{})
	if err := connection.WriteJSON(leave); err != nil {
		t.Fatalf("write leave: %v", err)
	}
}

func TestTwoClientsShareAuthoritativeMovement(t *testing.T) {
	cfg := testConfig()
	manager := room.NewManager(cfg.RoomTTL)
	created, _, err := manager.Ensure("EDGE-TEST-2")
	if err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	app := NewRouter(cfg, manager, slog.New(slog.NewTextHandler(io.Discard, nil)), func() bool { return true })
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	serverDone := make(chan error, 1)
	go func() { serverDone <- app.Listener(listener, fiber.ListenConfig{DisableStartupMessage: true}) }()
	t.Cleanup(func() {
		_ = manager.Close(context.Background())
		_ = app.Shutdown()
		<-serverDone
	})

	joinClient := func(displayName string) (*clientws.Conn, protocol.JoinedPayload) {
		t.Helper()
		header := http.Header{"Origin": []string{"http://127.0.0.1:3702"}}
		connection, _, dialErr := clientws.DefaultDialer.Dial("ws://"+listener.Addr().String()+"/ws/v1/rooms/"+created.Code(), header)
		if dialErr != nil {
			t.Fatalf("dial %s: %v", displayName, dialErr)
		}
		join, _ := protocol.NewEnvelope(protocol.TypeJoinRoom, "join-"+displayName, protocol.JoinRoomPayload{DisplayName: displayName})
		if writeErr := connection.WriteJSON(join); writeErr != nil {
			t.Fatalf("join %s: %v", displayName, writeErr)
		}
		var message protocol.Envelope
		if readErr := connection.ReadJSON(&message); readErr != nil {
			t.Fatalf("read joined %s: %v", displayName, readErr)
		}
		var joined protocol.JoinedPayload
		if decodeErr := message.DecodePayload(&joined); decodeErr != nil {
			t.Fatalf("decode joined %s: %v", displayName, decodeErr)
		}
		return connection, joined
	}

	first, firstJoined := joinClient("Umar")
	defer first.Close()
	second, _ := joinClient("Budi")
	defer second.Close()

	baseline := readSnapshot(t, second, 2)
	startX := snapshotPlayerX(t, baseline, firstJoined.PlayerID)
	input, _ := protocol.NewEnvelope(protocol.TypeInput, "", protocol.InputPayload{Sequence: 1, MoveX: 1, MoveY: 0})
	if err := first.WriteJSON(input); err != nil {
		t.Fatalf("write input: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		snapshot := readSnapshot(t, second, 2)
		if snapshotPlayerX(t, snapshot, firstJoined.PlayerID) > startX+5 {
			return
		}
	}
	t.Fatal("second client never observed first client's authoritative movement")
}

func readSnapshot(t *testing.T, connection *clientws.Conn, minimumPlayers int) protocol.SnapshotPayload {
	t.Helper()
	_ = connection.SetReadDeadline(time.Now().Add(2 * time.Second))
	for {
		var message protocol.Envelope
		if err := connection.ReadJSON(&message); err != nil {
			t.Fatalf("read snapshot: %v", err)
		}
		if message.Type != protocol.TypeSnapshot {
			continue
		}
		var snapshot protocol.SnapshotPayload
		if err := message.DecodePayload(&snapshot); err != nil {
			t.Fatalf("decode snapshot: %v", err)
		}
		if len(snapshot.Players) >= minimumPlayers {
			return snapshot
		}
	}
}

func snapshotPlayerX(t *testing.T, snapshot protocol.SnapshotPayload, playerID string) float64 {
	t.Helper()
	for _, player := range snapshot.Players {
		if player.ID == playerID {
			return player.X
		}
	}
	t.Fatalf("player %s missing from snapshot", playerID)
	return 0
}
