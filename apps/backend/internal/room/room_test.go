package room

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
	"unsafe"

	"survive-bro/apps/backend/internal/protocol"
)

func TestManagerCreatesInspectableRoomCode(t *testing.T) {
	manager := NewManager(time.Minute)
	t.Cleanup(func() { _ = manager.Close(context.Background()) })

	created, err := manager.Create()
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if len(created.Code()) != 6 {
		t.Fatalf("code length = %d", len(created.Code()))
	}
	if strings.ContainsAny(created.Code(), "0O1I") {
		t.Fatalf("code contains ambiguous character: %q", created.Code())
	}

	summary, err := manager.Inspect(context.Background(), strings.ToLower(created.Code()))
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if summary.State != StateLobby || summary.PlayerCount != 0 {
		t.Fatalf("unexpected summary: %#v", summary)
	}
}

func TestManagerEnsuresCanonicalNamedRoom(t *testing.T) {
	manager := NewManager(time.Minute)
	t.Cleanup(func() { _ = manager.Close(context.Background()) })

	first, created, err := manager.Ensure(" friday-squad ")
	if err != nil || !created {
		t.Fatalf("first Ensure() room=%v created=%v error=%v", first, created, err)
	}
	second, created, err := manager.Ensure("FRIDAY-SQUAD")
	if err != nil || created || first != second {
		t.Fatalf("second Ensure() same=%v created=%v error=%v", first == second, created, err)
	}
	if first.Code() != "FRIDAY-SQUAD" {
		t.Fatalf("canonical name = %q", first.Code())
	}
	if _, _, err := manager.Ensure("not allowed!"); !errors.Is(err, ErrInvalidRoomName) {
		t.Fatalf("invalid Ensure() error = %v", err)
	}
}

func TestNormalizeNameClonesBorrowedRequestMemory(t *testing.T) {
	source := []byte("BINARY-V2")
	borrowed := unsafe.String(&source[0], len(source))
	normalized, err := NormalizeName(borrowed)
	if err != nil {
		t.Fatalf("NormalizeName() error = %v", err)
	}
	source[0] = 'X'
	if normalized != "BINARY-V2" {
		t.Fatalf("normalized name changed with source buffer: %q", normalized)
	}
}

func TestRoomJoinCapacityAndHostTransfer(t *testing.T) {
	manager := NewManager(time.Minute)
	t.Cleanup(func() { _ = manager.Close(context.Background()) })
	created, err := manager.Create()
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	type joinedClient struct {
		result JoinResult
		send   chan protocol.Envelope
	}
	clients := make([]joinedClient, 0, MaxPlayers)
	for index := range MaxPlayers {
		send := make(chan protocol.Envelope, 32)
		result, joinErr := created.Join(context.Background(), " Player "+string(rune('A'+index))+" ", nil, send)
		if joinErr != nil {
			t.Fatalf("Join(%d) error = %v", index, joinErr)
		}
		if result.Host != (index == 0) {
			t.Fatalf("Join(%d) host = %v", index, result.Host)
		}
		clients = append(clients, joinedClient{result: result, send: send})
	}

	if _, err := created.Join(context.Background(), "Fifth", nil, make(chan protocol.Envelope, 4)); !errors.Is(err, ErrFull) {
		t.Fatalf("fifth Join() error = %v, want ErrFull", err)
	}
	if err := created.Leave(context.Background(), clients[0].result.PlayerID); err != nil {
		t.Fatalf("Leave(host) error = %v", err)
	}

	var latest protocol.RoomStatePayload
	for {
		select {
		case envelope := <-clients[1].send:
			if envelope.Type == protocol.TypeRoomState {
				if err := envelope.DecodePayload(&latest); err != nil {
					t.Fatalf("decode room state: %v", err)
				}
			}
		default:
			if latest.HostPlayerID != clients[1].result.PlayerID {
				t.Fatalf("host after transfer = %q, want %q", latest.HostPlayerID, clients[1].result.PlayerID)
			}
			return
		}
	}
}

func TestEmptyRoomExpires(t *testing.T) {
	manager := NewManager(15 * time.Millisecond)
	if _, err := manager.Create(); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	deadline := time.Now().Add(time.Second)
	for manager.Count() != 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if manager.Count() != 0 {
		t.Fatalf("Count() = %d, want expired room removed", manager.Count())
	}
}

func TestSlowClientIsPrunedAndRoomExpires(t *testing.T) {
	manager := NewManager(15 * time.Millisecond)
	created, err := manager.Create()
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Capacity one is occupied by joined, so the critical room_state broadcast
	// cannot be queued and the actor disconnects the slow client.
	if _, err := created.Join(context.Background(), "Slow", nil, make(chan protocol.Envelope, 1)); err != nil {
		t.Fatalf("Join() error = %v", err)
	}
	deadline := time.Now().Add(time.Second)
	for manager.Count() != 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if manager.Count() != 0 {
		t.Fatalf("Count() = %d, want pruned room to expire", manager.Count())
	}
}
