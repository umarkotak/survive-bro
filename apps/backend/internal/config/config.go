package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultHTTPAddress      = ":3701"
	defaultAllowedOrigin    = "http://localhost:3702,http://127.0.0.1:3702,https://survive-bro-dev.cabocil.com"
	defaultRoomTTL          = 10 * time.Minute
	defaultJoinTimeout      = 5 * time.Second
	defaultShutdownTimeout  = 10 * time.Second
	defaultHTTPBodyLimit    = 64 * 1024
	defaultWebSocketMsgSize = 16 * 1024
	defaultGameDataPath     = "../../game-data/game.json"
)

type Config struct {
	HTTPAddress             string
	AllowedOrigins          []string
	RoomTTL                 time.Duration
	JoinTimeout             time.Duration
	ShutdownTimeout         time.Duration
	HTTPBodyLimitBytes      int
	WebSocketMessageBytes   int64
	WebSocketCriticalBuffer int
	GameDataPath            string
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddress:             envOrDefault("HTTP_ADDRESS", defaultHTTPAddress),
		AllowedOrigins:          splitCSV(envOrDefault("ALLOWED_ORIGINS", defaultAllowedOrigin)),
		RoomTTL:                 defaultRoomTTL,
		JoinTimeout:             defaultJoinTimeout,
		ShutdownTimeout:         defaultShutdownTimeout,
		HTTPBodyLimitBytes:      defaultHTTPBodyLimit,
		WebSocketMessageBytes:   defaultWebSocketMsgSize,
		WebSocketCriticalBuffer: 64,
		GameDataPath:            envOrDefault("GAME_DATA_PATH", defaultGameDataPath),
	}

	var err error
	if cfg.RoomTTL, err = durationEnv("ROOM_TTL", cfg.RoomTTL); err != nil {
		return Config{}, err
	}
	if cfg.JoinTimeout, err = durationEnv("WS_JOIN_TIMEOUT", cfg.JoinTimeout); err != nil {
		return Config{}, err
	}
	if cfg.ShutdownTimeout, err = durationEnv("SHUTDOWN_TIMEOUT", cfg.ShutdownTimeout); err != nil {
		return Config{}, err
	}
	if cfg.HTTPBodyLimitBytes, err = positiveIntEnv("HTTP_BODY_LIMIT_BYTES", cfg.HTTPBodyLimitBytes); err != nil {
		return Config{}, err
	}
	if cfg.WebSocketMessageBytes, err = positiveInt64Env("WS_MESSAGE_LIMIT_BYTES", cfg.WebSocketMessageBytes); err != nil {
		return Config{}, err
	}
	if cfg.WebSocketCriticalBuffer, err = positiveIntEnv("WS_CRITICAL_BUFFER", cfg.WebSocketCriticalBuffer); err != nil {
		return Config{}, err
	}
	if len(cfg.AllowedOrigins) == 0 {
		return Config{}, fmt.Errorf("ALLOWED_ORIGINS must contain at least one origin")
	}

	return cfg, nil
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func durationEnv(key string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration: %q", key, value)
	}
	return parsed, nil
}

func positiveIntEnv(key string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer: %q", key, value)
	}
	return parsed, nil
}

func positiveInt64Env(key string, fallback int64) (int64, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer: %q", key, value)
	}
	return parsed, nil
}
