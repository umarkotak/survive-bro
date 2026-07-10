package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	clientws "github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v3"

	"survive-bro/apps/backend/internal/config"
	"survive-bro/apps/backend/internal/protocol"
	"survive-bro/apps/backend/internal/room"
)

func testConfig() config.Config {
	return config.Config{
		HTTPAddress:             ":0",
		AllowedOrigins:          []string{"http://localhost:3702", "http://127.0.0.1:3702", "https://survive-bro-dev.cabocil.com"},
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

func TestCORSAllowsAnyOrigin(t *testing.T) {
	app, _ := testRouter(t, func() bool { return true })
	request := httptest.NewRequest(http.MethodOptions, "/api/v1/rooms/FRIDAY-SQUAD", http.NoBody)
	request.Header.Set("Origin", "https://survive-bro-dev.cabocil.com")
	request.Header.Set("Access-Control-Request-Method", http.MethodPut)

	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("OPTIONS room error = %v", err)
	}
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("OPTIONS room status = %d", response.StatusCode)
	}
	if got := response.Header.Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
	if got := response.Header.Get("Access-Control-Allow-Methods"); !strings.Contains(got, http.MethodPut) {
		t.Fatalf("Access-Control-Allow-Methods = %q", got)
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
	decodeHTTPJSON(t, createResponse.Body, &created)

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
	decodeHTTPJSON(t, inspectResponse.Body, &inspected)
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
		decodeHTTPJSON(t, response.Body, &ensured)
		if ensured.RoomName != "FRIDAY-SQUAD" || ensured.Created != (index == 0) {
			t.Fatalf("unexpected ensure response: %#v", ensured)
		}
	}
}

func TestWebSocketJoinAndPing(t *testing.T) {
	cfg := testConfig()
	manager := room.NewManager(cfg.RoomTTL)
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
	roomName := "BINARY-HTTP"
	ensureRequest, err := http.NewRequest(http.MethodPut, "http://"+listener.Addr().String()+"/api/v1/rooms/"+roomName, http.NoBody)
	if err != nil {
		t.Fatalf("create ensure request: %v", err)
	}
	ensureResponse, err := http.DefaultClient.Do(ensureRequest)
	if err != nil {
		t.Fatalf("ensure room: %v", err)
	}
	ensureResponse.Body.Close()
	if ensureResponse.StatusCode != http.StatusOK {
		t.Fatalf("ensure room status = %d", ensureResponse.StatusCode)
	}

	header := http.Header{"Origin": []string{"https://survive-bro-dev.cabocil.com"}}
	connection, response, err := clientws.DefaultDialer.Dial("ws://"+listener.Addr().String()+"/ws/v2/rooms/"+roomName, header)
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
	writeWebSocketEnvelope(t, connection, join)

	joined := readWebSocketEnvelope(t, connection)
	if joined.Type != protocol.TypeJoined {
		t.Fatalf("first message type = %q", joined.Type)
	}
	var joinedPayload protocol.JoinedPayload
	if err := joined.DecodePayload(&joinedPayload); err != nil {
		t.Fatalf("decode joined: %v", err)
	}
	if !joinedPayload.Host || joinedPayload.RoomName != roomName || joinedPayload.ReconnectToken == "" {
		t.Fatalf("unexpected joined payload: %#v", joinedPayload)
	}

	seen := map[protocol.MessageType]bool{}
	for !(seen[protocol.TypeMatchStarted] && seen[protocol.TypeRoomState] && seen[protocol.TypeSnapshot]) {
		message := readWebSocketEnvelope(t, connection)
		seen[message.Type] = true
	}

	ping, err := protocol.NewEnvelope(protocol.TypePing, "ping-1", struct{}{})
	if err != nil {
		t.Fatalf("ping envelope: %v", err)
	}
	writeWebSocketEnvelope(t, connection, ping)
	for {
		pong := readWebSocketEnvelope(t, connection)
		if pong.Type == protocol.TypePong {
			if pong.RequestID != "ping-1" {
				t.Fatalf("unexpected pong: %#v", pong)
			}
			break
		}
	}

	leave, _ := protocol.NewEnvelope(protocol.TypeLeaveRoom, "leave-1", struct{}{})
	writeWebSocketEnvelope(t, connection, leave)
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
		connection, _, dialErr := clientws.DefaultDialer.Dial("ws://"+listener.Addr().String()+"/ws/v2/rooms/"+created.Code(), header)
		if dialErr != nil {
			t.Fatalf("dial %s: %v", displayName, dialErr)
		}
		join, _ := protocol.NewEnvelope(protocol.TypeJoinRoom, "join-"+displayName, protocol.JoinRoomPayload{DisplayName: displayName})
		writeWebSocketEnvelope(t, connection, join)
		message := readWebSocketEnvelope(t, connection)
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
	writeWebSocketEnvelope(t, first, input)

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
		message := readWebSocketEnvelope(t, connection)
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

func writeWebSocketEnvelope(t *testing.T, connection *clientws.Conn, envelope protocol.Envelope) {
	t.Helper()
	encoded, err := protocol.Encode(envelope)
	if err != nil {
		t.Fatalf("encode WebSocket envelope: %v", err)
	}
	if err := connection.WriteMessage(clientws.BinaryMessage, encoded); err != nil {
		t.Fatalf("write WebSocket envelope: %v", err)
	}
}

func readWebSocketEnvelope(t *testing.T, connection *clientws.Conn) protocol.Envelope {
	t.Helper()
	messageType, data, err := connection.ReadMessage()
	if err != nil {
		t.Fatalf("read WebSocket envelope: %v", err)
	}
	if messageType != clientws.BinaryMessage {
		t.Fatalf("WebSocket message type = %d, want binary", messageType)
	}
	envelope, err := protocol.Decode(data)
	if err != nil {
		t.Fatalf("decode WebSocket envelope: %v", err)
	}
	return envelope
}

func decodeHTTPJSON(t *testing.T, reader io.Reader, target any) {
	t.Helper()
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read HTTP JSON: %v", err)
	}
	if err := sonic.Unmarshal(data, target); err != nil {
		t.Fatalf("decode HTTP JSON: %v", err)
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
