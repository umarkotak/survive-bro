package config

import (
	"reflect"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("HTTP_ADDRESS", "")
	t.Setenv("ALLOWED_ORIGINS", "")
	t.Setenv("ROOM_TTL", "")
	t.Setenv("WS_JOIN_TIMEOUT", "")
	t.Setenv("SHUTDOWN_TIMEOUT", "")
	t.Setenv("HTTP_BODY_LIMIT_BYTES", "")
	t.Setenv("WS_MESSAGE_LIMIT_BYTES", "")
	t.Setenv("WS_CRITICAL_BUFFER", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTPAddress != ":3701" {
		t.Fatalf("HTTPAddress = %q", cfg.HTTPAddress)
	}
	if !reflect.DeepEqual(cfg.AllowedOrigins, []string{"http://localhost:3702", "http://127.0.0.1:3702"}) {
		t.Fatalf("AllowedOrigins = %#v", cfg.AllowedOrigins)
	}
	if cfg.RoomTTL != 10*time.Minute || cfg.JoinTimeout != 5*time.Second {
		t.Fatalf("unexpected durations: room=%s join=%s", cfg.RoomTTL, cfg.JoinTimeout)
	}
}

func TestLoadRejectsInvalidPositiveValues(t *testing.T) {
	t.Setenv("WS_MESSAGE_LIMIT_BYTES", "0")
	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want invalid message limit error")
	}
}

func TestLoadParsesOrigins(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "https://game.example, http://localhost:5173")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	want := []string{"https://game.example", "http://localhost:5173"}
	if !reflect.DeepEqual(cfg.AllowedOrigins, want) {
		t.Fatalf("AllowedOrigins = %#v, want %#v", cfg.AllowedOrigins, want)
	}
}
