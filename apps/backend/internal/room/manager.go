package room

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"survive-bro/apps/backend/internal/observability"
)

const roomCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

var ErrNotFound = errors.New("room not found")
var ErrInvalidRoomName = errors.New("room name must contain 1 to 24 letters, numbers, hyphens, or underscores")

type Manager struct {
	mu      sync.RWMutex
	rooms   map[string]*Room
	ttl     time.Duration
	now     func() time.Time
	metrics *observability.Collector
}

func NewManager(ttl time.Duration) *Manager {
	return &Manager{
		rooms:   make(map[string]*Room),
		ttl:     ttl,
		now:     time.Now,
		metrics: observability.NewCollector(),
	}
}

func (m *Manager) Create() (*Room, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for range 32 {
		code, err := secureRoomCode()
		if err != nil {
			return nil, err
		}
		if _, exists := m.rooms[code]; exists {
			continue
		}
		id, err := secureToken("room_", 16)
		if err != nil {
			return nil, err
		}
		created := newRoom(id, code, m.ttl, m.now, m.metrics, m.remove)
		m.rooms[code] = created
		return created, nil
	}
	return nil, fmt.Errorf("generate unique room code")
}

func (m *Manager) Ensure(name string) (*Room, bool, error) {
	normalized, err := NormalizeName(name)
	if err != nil {
		return nil, false, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.rooms[normalized]; ok {
		return existing, false, nil
	}
	id, err := secureToken("room_", 16)
	if err != nil {
		return nil, false, err
	}
	created := newRoom(id, normalized, m.ttl, m.now, m.metrics, m.remove)
	m.rooms[normalized] = created
	return created, true, nil
}

func (m *Manager) Find(code string) (*Room, error) {
	normalized, err := NormalizeName(code)
	if err != nil {
		return nil, ErrNotFound
	}
	m.mu.RLock()
	room, ok := m.rooms[normalized]
	m.mu.RUnlock()
	if !ok {
		return nil, ErrNotFound
	}
	return room, nil
}

func NormalizeName(name string) (string, error) {
	// Fiber path parameters may reference a request buffer that is reused after
	// the handler returns. Clone the canonical name before storing it as a map key.
	normalized := strings.Clone(strings.ToUpper(strings.TrimSpace(name)))
	if len(normalized) < 1 || len(normalized) > 24 {
		return "", ErrInvalidRoomName
	}
	for _, character := range normalized {
		if (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') || character == '-' || character == '_' {
			continue
		}
		return "", ErrInvalidRoomName
	}
	return normalized, nil
}

func (m *Manager) Inspect(ctx context.Context, code string) (Summary, error) {
	room, err := m.Find(code)
	if err != nil {
		return Summary{}, err
	}
	return room.Summary(ctx)
}

func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.rooms)
}

func (m *Manager) Metrics() *observability.Collector { return m.metrics }

func (m *Manager) Close(ctx context.Context) error {
	m.mu.RLock()
	rooms := make([]*Room, 0, len(m.rooms))
	for _, room := range m.rooms {
		rooms = append(rooms, room)
	}
	m.mu.RUnlock()

	for _, room := range rooms {
		if err := room.Close(ctx); err != nil && !errors.Is(err, ErrClosed) {
			return err
		}
	}
	return nil
}

func (m *Manager) remove(code string, target *Room) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if current, ok := m.rooms[code]; ok && current == target {
		delete(m.rooms, code)
		m.metrics.RemoveRoom(code)
	}
}

func secureRoomCode() (string, error) {
	buffer := make([]byte, 6)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	for index, value := range buffer {
		buffer[index] = roomCodeAlphabet[int(value)%len(roomCodeAlphabet)]
	}
	return string(buffer), nil
}
