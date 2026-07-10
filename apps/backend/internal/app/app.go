package app

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"

	"github.com/gofiber/fiber/v3"

	"survive-bro/apps/backend/internal/config"
	"survive-bro/apps/backend/internal/room"
	"survive-bro/apps/backend/internal/transport/httpapi"
)

type Application struct {
	HTTP  *fiber.App
	Rooms *room.Manager
	ready atomic.Bool
}

func New(cfg config.Config, logger *slog.Logger) *Application {
	application := &Application{Rooms: room.NewManager(cfg.RoomTTL)}
	application.ready.Store(true)
	application.HTTP = httpapi.NewRouter(cfg, application.Rooms, logger, application.ready.Load)
	return application
}

func (a *Application) MarkDraining() {
	a.ready.Store(false)
}

func (a *Application) Shutdown(ctx context.Context) error {
	a.MarkDraining()
	roomErr := a.Rooms.Close(ctx)
	httpErr := a.HTTP.ShutdownWithContext(ctx)
	return errors.Join(roomErr, httpErr)
}
