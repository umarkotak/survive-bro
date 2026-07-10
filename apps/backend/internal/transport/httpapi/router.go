package httpapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	recoverer "github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"

	"survive-bro/apps/backend/internal/config"
	"survive-bro/apps/backend/internal/protocol"
	"survive-bro/apps/backend/internal/room"
)

const roomLocalKey = "survive_bro_room"

type ReadyFunc func() bool

type Handler struct {
	rooms  *room.Manager
	cfg    config.Config
	logger *slog.Logger
	ready  ReadyFunc
}

func NewRouter(cfg config.Config, rooms *room.Manager, logger *slog.Logger, ready ReadyFunc) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:     "Survive Bro Game Server",
		BodyLimit:   cfg.HTTPBodyLimitBytes,
		JSONEncoder: sonic.Marshal,
		JSONDecoder: sonic.Unmarshal,
	})
	handler := &Handler{rooms: rooms, cfg: cfg, logger: logger, ready: ready}

	app.Use(requestid.New())
	app.Use(recoverer.New())
	app.Use(handler.logRequest)
	app.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
	}))

	app.Get("/health/live", handler.live)
	app.Get("/health/ready", handler.readiness)
	app.Get("/metrics", handler.metrics)
	app.Post("/api/v1/rooms", handler.createRoom)
	app.Put("/api/v1/rooms/:roomName", handler.ensureRoom)
	app.Get("/api/v1/rooms/:roomName", handler.inspectRoom)
	app.Get("/ws/v2/rooms/:roomName", handler.prepareWebSocket, websocket.New(handler.serveWebSocket, websocket.Config{
		HandshakeTimeout:  cfg.JoinTimeout,
		Origins:           cfg.AllowedOrigins,
		AllowEmptyOrigin:  false,
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		EnableCompression: false,
		RecoverHandler: func(connection *websocket.Conn) {
			if recovered := recover(); recovered != nil {
				logger.Error("websocket panic", "error", recovered, "room_name", connection.Params("roomName"))
				_ = connection.Close()
			}
		},
	}))

	app.Use(func(c fiber.Ctx) error {
		return writeHTTPError(c, fiber.StatusNotFound, "not_found", "route not found")
	})

	return app
}

func (h *Handler) logRequest(c fiber.Ctx) error {
	started := time.Now()
	err := c.Next()
	h.logger.Info("http_request",
		"method", c.Method(),
		"path", c.Path(),
		"status", c.Response().StatusCode(),
		"duration_ms", time.Since(started).Milliseconds(),
		"request_id", c.Get(fiber.HeaderXRequestID),
	)
	return err
}

func (h *Handler) live(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *Handler) readiness(c fiber.Ctx) error {
	if !h.ready() {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "draining"})
	}
	return c.JSON(fiber.Map{"status": "ready"})
}

func (h *Handler) metrics(c fiber.Ctx) error {
	c.Set(fiber.HeaderContentType, "text/plain; version=0.0.4; charset=utf-8")
	return c.SendString(fmt.Sprintf("# TYPE survive_bro_active_rooms gauge\nsurvive_bro_active_rooms %d\n", h.rooms.Count()))
}

func (h *Handler) createRoom(c fiber.Ctx) error {
	created, err := h.rooms.Create()
	if err != nil {
		h.logger.Error("create room", "error", err)
		return writeHTTPError(c, fiber.StatusInternalServerError, "internal_error", "could not create room")
	}
	summary, err := created.Summary(c.Context())
	if err != nil {
		h.logger.Error("inspect created room", "error", err, "room_code", created.Code())
		return writeHTTPError(c, fiber.StatusInternalServerError, "internal_error", "could not create room")
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"roomName":         summary.Code,
		"expiresInSeconds": max(0, int(math.Ceil(time.Until(summary.ExpiresAt).Seconds()))),
	})
}

func (h *Handler) ensureRoom(c fiber.Ctx) error {
	ensured, created, err := h.rooms.Ensure(c.Params("roomName"))
	if errors.Is(err, room.ErrInvalidRoomName) {
		return writeHTTPError(c, fiber.StatusBadRequest, "invalid_room_name", err.Error())
	}
	if err != nil {
		h.logger.Error("ensure room", "error", err)
		return writeHTTPError(c, fiber.StatusInternalServerError, "internal_error", "could not ensure room")
	}
	summary, err := ensured.Summary(c.Context())
	if err != nil {
		return writeHTTPError(c, fiber.StatusInternalServerError, "internal_error", "could not inspect room")
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"roomName": summary.Code,
		"status":   summary.State,
		"created":  created,
	})
}

func (h *Handler) inspectRoom(c fiber.Ctx) error {
	summary, err := h.rooms.Inspect(c.Context(), c.Params("roomName"))
	if errors.Is(err, room.ErrNotFound) || errors.Is(err, room.ErrClosed) {
		return writeHTTPError(c, fiber.StatusNotFound, "room_not_found", "room does not exist")
	}
	if err != nil {
		h.logger.Error("inspect room", "error", err, "room_name", c.Params("roomName"))
		return writeHTTPError(c, fiber.StatusInternalServerError, "internal_error", "could not inspect room")
	}
	return c.JSON(fiber.Map{
		"roomName":    summary.Code,
		"status":      summary.State,
		"playerCount": summary.PlayerCount,
		"maxPlayers":  room.MaxPlayers,
		"joinable":    summary.PlayerCount < room.MaxPlayers,
	})
}

func (h *Handler) prepareWebSocket(c fiber.Ctx) error {
	if !websocket.IsWebSocketUpgrade(c) {
		return writeHTTPError(c, fiber.StatusUpgradeRequired, "upgrade_required", "WebSocket upgrade required")
	}
	h.logger.Info("websocket_upgrade", "room_name", c.Params("roomName"), "origin", c.Get(fiber.HeaderOrigin))
	currentRoom, err := h.rooms.Find(c.Params("roomName"))
	if errors.Is(err, room.ErrNotFound) {
		return writeHTTPError(c, fiber.StatusNotFound, "room_not_found", "room does not exist")
	}
	if err != nil {
		return writeHTTPError(c, fiber.StatusInternalServerError, "internal_error", "could not inspect room")
	}
	c.Locals(roomLocalKey, currentRoom)
	return c.Next()
}

func (h *Handler) serveWebSocket(connection *websocket.Conn) {
	currentRoom, ok := connection.Locals(roomLocalKey).(*room.Room)
	if !ok || currentRoom == nil {
		h.writeAndClose(connection, protocol.Error("", "internal_error", "room context is unavailable"), websocket.CloseInternalServerErr)
		return
	}

	connection.SetReadLimit(h.cfg.WebSocketMessageBytes)
	_ = connection.SetReadDeadline(time.Now().Add(h.cfg.JoinTimeout))

	first, err := readBinaryEnvelope(connection)
	if err != nil {
		h.writeAndClose(connection, protocol.Error("", "join_required", "join_room must be sent before the deadline"), websocket.ClosePolicyViolation)
		return
	}
	if first.Type != protocol.TypeJoinRoom {
		h.writeAndClose(connection, protocol.Error(first.RequestID, "join_required", "first message must be join_room"), websocket.ClosePolicyViolation)
		return
	}
	var payload protocol.JoinRoomPayload
	if err := first.DecodePayload(&payload); err != nil {
		h.writeAndClose(connection, protocol.Error(first.RequestID, "invalid_payload", "join_room payload is invalid"), websocket.CloseInvalidFramePayloadData)
		return
	}

	outgoing := make(chan protocol.Envelope, h.cfg.WebSocketCriticalBuffer)
	joined, err := currentRoom.Join(context.Background(), payload.DisplayName, payload.ReconnectToken, outgoing)
	if err != nil {
		code, message := joinError(err)
		h.writeAndClose(connection, protocol.Error(first.RequestID, code, message), websocket.ClosePolicyViolation)
		return
	}

	_ = connection.SetReadDeadline(time.Now().Add(30 * time.Second))
	writerDone := make(chan struct{})
	go h.writeLoop(connection, outgoing, writerDone)

	defer func() {
		leaveContext, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = currentRoom.Leave(leaveContext, joined.PlayerID)
		<-writerDone
	}()

	for {
		message, err := readBinaryEnvelope(connection)
		if err != nil {
			return
		}
		_ = connection.SetReadDeadline(time.Now().Add(30 * time.Second))
		if err := currentRoom.Handle(context.Background(), joined.PlayerID, message); err != nil {
			return
		}
		if message.Type == protocol.TypeLeaveRoom {
			return
		}
	}
}

func (h *Handler) writeLoop(connection *websocket.Conn, outgoing <-chan protocol.Envelope, done chan<- struct{}) {
	defer close(done)
	defer connection.Close()
	for message := range outgoing {
		_ = connection.SetWriteDeadline(time.Now().Add(5 * time.Second))
		encoded, err := protocol.Encode(message)
		if err != nil {
			h.logger.Error("encode_websocket_message", "error", err, "message_type", message.Type)
			return
		}
		if err := connection.WriteMessage(websocket.BinaryMessage, encoded); err != nil {
			return
		}
	}
}

func (h *Handler) writeAndClose(connection *websocket.Conn, message protocol.Envelope, closeCode int) {
	_ = connection.SetWriteDeadline(time.Now().Add(2 * time.Second))
	if encoded, err := protocol.Encode(message); err == nil {
		_ = connection.WriteMessage(websocket.BinaryMessage, encoded)
	}
	_ = connection.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(closeCode, "connection closed"), time.Now().Add(2*time.Second))
}

func readBinaryEnvelope(connection *websocket.Conn) (protocol.Envelope, error) {
	messageType, data, err := connection.ReadMessage()
	if err != nil {
		return protocol.Envelope{}, err
	}
	if messageType != websocket.BinaryMessage {
		return protocol.Envelope{}, fmt.Errorf("WebSocket application messages must be binary")
	}
	return protocol.Decode(data)
}

func joinError(err error) (string, string) {
	switch {
	case errors.Is(err, room.ErrInvalidDisplayName):
		return "invalid_display_name", "display name must contain 1 to 20 characters"
	case errors.Is(err, room.ErrFull):
		return "room_full", "room is full"
	case errors.Is(err, room.ErrReconnectNotImplemented):
		return "reconnect_unavailable", "reconnection is not available in the current backend milestone"
	case errors.Is(err, room.ErrClosed):
		return "room_not_found", "room does not exist"
	default:
		return "internal_error", "could not join room"
	}
}

func writeHTTPError(c fiber.Ctx, status int, code, message string) error {
	return c.Status(status).JSON(fiber.Map{"error": fiber.Map{"code": code, "message": message}})
}
