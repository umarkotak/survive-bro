package protocol

import (
	"fmt"
	"reflect"
)

const Version uint8 = 2

type MessageType uint8

const (
	TypeJoinRoom          MessageType = 1
	TypeLeaveRoom         MessageType = 2
	TypePing              MessageType = 3
	TypeInput             MessageType = 4
	TypeJoined            MessageType = 64
	TypeRoomState         MessageType = 65
	TypeMatchStarted      MessageType = 66
	TypeSnapshot          MessageType = 67
	TypeProjectileSpawned MessageType = 68
	TypeProjectileRemoved MessageType = 69
	TypeMatchEnded        MessageType = 70
	TypePong              MessageType = 71
	TypeUpgradeApplied    MessageType = 76
	TypeError             MessageType = 126
	TypeServerClosed      MessageType = 127
)

type Envelope struct {
	Version   uint8
	Type      MessageType
	RequestID string
	Payload   any
}

type JoinRoomPayload struct {
	DisplayName    string  `json:"displayName"`
	CharacterID    string  `json:"characterId"`
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
	RoomName    string        `json:"roomName"`
	MapID       string        `json:"mapId"`
	MapWidth    float64       `json:"mapWidth"`
	MapHeight   float64       `json:"mapHeight"`
	StartedAtMs int64         `json:"startedAtMs"`
	Obstacles   []Obstacle    `json:"obstacles"`
	DurationMs  int64         `json:"durationMs"`
	Events      []SystemEvent `json:"events"`
}

type SystemEvent struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
	AtMs        int64  `json:"atMs"`
}

type SnapshotPlayer struct {
	ID                 string  `json:"id"`
	DisplayName        string  `json:"displayName"`
	CharacterID        string  `json:"characterId"`
	X                  float64 `json:"x"`
	Y                  float64 `json:"y"`
	VelocityX          float64 `json:"velocityX"`
	VelocityY          float64 `json:"velocityY"`
	MovementSpeed      float64 `json:"movementSpeed"`
	ArmorPercent       float64 `json:"armorPercent"`
	HealthRegeneration float64 `json:"healthRegeneration"`
	AttackBuffPercent  float64 `json:"attackBuffPercent"`
	CooldownPercent    float64 `json:"cooldownPercent"`
	SpellDamage        int     `json:"spellDamage"`
	ProjectileSpeed    float64 `json:"projectileSpeed"`
	SpellBurst         int     `json:"spellBurst"`
	SpellDirections    int     `json:"spellDirections"`
	Facing             string  `json:"facing"`
	HP                 int     `json:"hp"`
	MaxHP              int     `json:"maxHp"`
	Alive              bool    `json:"alive"`
	LastProcessedInput uint64  `json:"lastProcessedInput"`
	Kills              int     `json:"kills"`
}

type SnapshotMonster struct {
	ID     uint64  `json:"id"`
	TypeID string  `json:"typeId"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	HP     int     `json:"hp"`
	MaxHP  int     `json:"maxHp"`
}

type SnapshotPickup struct {
	ID   uint64  `json:"id"`
	Kind string  `json:"kind"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
}

type SnapshotTeam struct {
	Level              int     `json:"level"`
	Experience         int     `json:"experience"`
	ExperienceRequired int     `json:"experienceRequired"`
	TotalKills         int     `json:"totalKills"`
	ProjectileCount    int     `json:"projectileCount,omitempty"`
	PickupRadius       float64 `json:"pickupRadius,omitempty"`
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
	Score      int    `json:"score"`
}

type UpgradeAppliedPayload struct {
	PlayerID   string  `json:"playerId"`
	Source     string  `json:"source"`
	Attribute  string  `json:"attribute"`
	BaseValue  float64 `json:"baseValue"`
	AddedValue float64 `json:"addedValue"`
	FinalValue float64 `json:"finalValue"`
}

func NewEnvelope(messageType MessageType, requestID string, payload any) (Envelope, error) {
	if !messageType.valid() {
		return Envelope{}, fmt.Errorf("unknown message type %d", messageType)
	}
	if len(requestID) > 255 {
		return Envelope{}, fmt.Errorf("request ID exceeds 255 bytes")
	}
	return Envelope{Version: Version, Type: messageType, RequestID: requestID, Payload: payload}, nil
}

func Error(requestID, code, message string) Envelope {
	envelope, _ := NewEnvelope(TypeError, requestID, ErrorPayload{Code: code, Message: message})
	return envelope
}

func (e Envelope) DecodePayload(target any) error {
	if target == nil || e.Payload == nil {
		return fmt.Errorf("payload is required")
	}
	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Pointer || targetValue.IsNil() {
		return fmt.Errorf("payload target must be a non-nil pointer")
	}
	payloadValue := reflect.ValueOf(e.Payload)
	if !payloadValue.Type().AssignableTo(targetValue.Elem().Type()) {
		return fmt.Errorf("payload type %T cannot decode into %T", e.Payload, target)
	}
	targetValue.Elem().Set(payloadValue)
	return nil
}

func (t MessageType) valid() bool {
	switch t {
	case TypeJoinRoom, TypeLeaveRoom, TypePing, TypeInput, TypeJoined, TypeRoomState,
		TypeMatchStarted, TypeSnapshot, TypeProjectileSpawned, TypeProjectileRemoved,
		TypeMatchEnded, TypePong, TypeUpgradeApplied, TypeError, TypeServerClosed:
		return true
	default:
		return false
	}
}
