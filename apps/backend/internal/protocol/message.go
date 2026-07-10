package protocol

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const Version = 1

const (
	TypeJoinRoom          = "join_room"
	TypeLeaveRoom         = "leave_room"
	TypePing              = "ping"
	TypeInput             = "input"
	TypeJoined            = "joined"
	TypeRoomState         = "room_state"
	TypeMatchStarted      = "match_started"
	TypeSnapshot          = "snapshot"
	TypeProjectileSpawned = "projectile_spawned"
	TypeProjectileRemoved = "projectile_removed"
	TypeMatchEnded        = "match_ended"
	TypePong              = "pong"
	TypeError             = "error"
	TypeServerClosed      = "server_shutdown"
)

type Envelope struct {
	Version   int             `json:"v"`
	Type      string          `json:"type"`
	RequestID string          `json:"requestId,omitempty"`
	Payload   json.RawMessage `json:"payload"`
}

type JoinRoomPayload struct {
	DisplayName    string  `json:"displayName"`
	ReconnectToken *string `json:"reconnectToken"`
}

type JoinedPayload struct {
	PlayerID       string `json:"playerId"`
	ReconnectToken string `json:"reconnectToken"`
	RoomName       string `json:"roomName"`
	Host           bool   `json:"host"`
}

type InputPayload struct {
	Sequence uint64  `json:"sequence"`
	MoveX    float64 `json:"moveX"`
	MoveY    float64 `json:"moveY"`
}

type PlayerState struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	CharacterID string `json:"characterId,omitempty"`
	Ready       bool   `json:"ready"`
	Connected   bool   `json:"connected"`
}

type RoomStatePayload struct {
	Status       string        `json:"status"`
	HostPlayerID string        `json:"hostPlayerId,omitempty"`
	Players      []PlayerState `json:"players"`
}

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ServerShutdownPayload struct {
	Reason string `json:"reason"`
}

type Obstacle struct {
	ID     string  `json:"id"`
	Type   string  `json:"type"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Radius float64 `json:"radius"`
}

type MatchStartedPayload struct {
	RoomName    string     `json:"roomName"`
	MapID       string     `json:"mapId"`
	MapWidth    float64    `json:"mapWidth"`
	MapHeight   float64    `json:"mapHeight"`
	StartedAtMs int64      `json:"startedAtMs"`
	Obstacles   []Obstacle `json:"obstacles"`
}

type SnapshotPlayer struct {
	ID                 string  `json:"id"`
	DisplayName        string  `json:"displayName"`
	X                  float64 `json:"x"`
	Y                  float64 `json:"y"`
	VelocityX          float64 `json:"velocityX"`
	VelocityY          float64 `json:"velocityY"`
	Facing             string  `json:"facing"`
	HP                 int     `json:"hp"`
	MaxHP              int     `json:"maxHp"`
	Alive              bool    `json:"alive"`
	LastProcessedInput uint64  `json:"lastProcessedInput"`
	Kills              int     `json:"kills"`
}

type SnapshotMonster struct {
	ID    uint64  `json:"id"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	HP    int     `json:"hp"`
	MaxHP int     `json:"maxHp"`
}

type SnapshotPickup struct {
	ID uint64  `json:"id"`
	X  float64 `json:"x"`
	Y  float64 `json:"y"`
}

type SnapshotTeam struct {
	Level              int `json:"level"`
	Experience         int `json:"experience"`
	ExperienceRequired int `json:"experienceRequired"`
	TotalKills         int `json:"totalKills"`
}

type SnapshotPayload struct {
	Tick         uint64            `json:"tick"`
	ServerTimeMs int64             `json:"serverTimeMs"`
	Players      []SnapshotPlayer  `json:"players"`
	Monsters     []SnapshotMonster `json:"monsters"`
	Pickups      []SnapshotPickup  `json:"pickups"`
	Team         SnapshotTeam      `json:"team"`
	RemainingMs  int64             `json:"remainingMs"`
}

type ProjectileSpawnedPayload struct {
	ProjectileID uint64  `json:"projectileId"`
	OwnerID      string  `json:"ownerId"`
	WeaponID     string  `json:"weaponId"`
	X            float64 `json:"x"`
	Y            float64 `json:"y"`
	VelocityX    float64 `json:"velocityX"`
	VelocityY    float64 `json:"velocityY"`
	SpawnTick    uint64  `json:"spawnTick"`
}

type ProjectileRemovedPayload struct {
	ProjectileID uint64 `json:"projectileId"`
	Reason       string `json:"reason"`
}

type MatchEndedPayload struct {
	Outcome    string `json:"outcome"`
	SurvivalMs int64  `json:"survivalMs"`
	TeamLevel  int    `json:"teamLevel"`
	TotalKills int    `json:"totalKills"`
}

func NewEnvelope(messageType, requestID string, payload any) (Envelope, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return Envelope{}, err
	}
	return Envelope{
		Version:   Version,
		Type:      messageType,
		RequestID: requestID,
		Payload:   raw,
	}, nil
}

func Error(requestID, code, message string) Envelope {
	envelope, _ := NewEnvelope(TypeError, requestID, ErrorPayload{Code: code, Message: message})
	return envelope
}

func (e Envelope) DecodePayload(target any) error {
	if len(e.Payload) == 0 {
		return fmt.Errorf("payload is required")
	}
	decoder := json.NewDecoder(bytes.NewReader(e.Payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("payload contains trailing data")
	}
	return nil
}
