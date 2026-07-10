package protocol

import (
	"encoding/hex"
	"math"
	"testing"

	"github.com/bytedance/sonic"
)

func TestInputGoldenFrame(t *testing.T) {
	envelope, err := NewEnvelope(TypeInput, "", InputPayload{Sequence: 154, MoveX: 0.5, MoveY: -1})
	if err != nil {
		t.Fatalf("NewEnvelope() error = %v", err)
	}
	encoded, err := Encode(envelope)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	if got, want := hex.EncodeToString(encoded), "0204009a0000000000003f000080bf"; got != want {
		t.Fatalf("frame = %s, want %s", got, want)
	}

	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	var payload InputPayload
	if err := decoded.DecodePayload(&payload); err != nil {
		t.Fatalf("DecodePayload() error = %v", err)
	}
	if payload.Sequence != 154 || payload.MoveX != 0.5 || payload.MoveY != -1 {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestJoinRoomRoundTrip(t *testing.T) {
	token := "reconnect-secret"
	envelope, _ := NewEnvelope(TypeJoinRoom, "join-1", JoinRoomPayload{DisplayName: "Ümar", ReconnectToken: &token})
	encoded, err := Encode(envelope)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	var payload JoinRoomPayload
	if err := decoded.DecodePayload(&payload); err != nil {
		t.Fatalf("DecodePayload() error = %v", err)
	}
	if decoded.RequestID != "join-1" || payload.DisplayName != "Ümar" || payload.ReconnectToken == nil || *payload.ReconnectToken != token {
		t.Fatalf("decoded = %#v payload=%#v", decoded, payload)
	}
}

func TestSnapshotRoundTrip(t *testing.T) {
	snapshot := benchmarkSnapshot()
	envelope, _ := NewEnvelope(TypeSnapshot, "", snapshot)
	encoded, err := Encode(envelope)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	var result SnapshotPayload
	if err := decoded.DecodePayload(&result); err != nil {
		t.Fatalf("DecodePayload() error = %v", err)
	}
	if len(result.Players) != 4 || len(result.Monsters) != 150 || len(result.Pickups) != 50 {
		t.Fatalf("unexpected entity counts: players=%d monsters=%d pickups=%d", len(result.Players), len(result.Monsters), len(result.Pickups))
	}
	if math.Abs(result.Players[0].X-snapshot.Players[0].X) > 0.001 || result.Team != snapshot.Team {
		t.Fatalf("snapshot mismatch: %#v", result)
	}
}

func TestBinarySnapshotIsSmallerThanSonicJSON(t *testing.T) {
	snapshot := benchmarkSnapshot()
	envelope, _ := NewEnvelope(TypeSnapshot, "", snapshot)
	binaryFrame, err := Encode(envelope)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	jsonFrame, err := sonic.Marshal(snapshot)
	if err != nil {
		t.Fatalf("sonic.Marshal() error = %v", err)
	}
	if len(binaryFrame) >= len(jsonFrame) {
		t.Fatalf("binary frame = %d bytes, Sonic JSON = %d bytes", len(binaryFrame), len(jsonFrame))
	}
	t.Logf("binary=%d bytes sonic_json=%d bytes reduction=%.1f%%", len(binaryFrame), len(jsonFrame), 100*(1-float64(len(binaryFrame))/float64(len(jsonFrame))))
}

func TestDecodeRejectsMalformedFrames(t *testing.T) {
	for _, frame := range [][]byte{
		{},
		{1, byte(TypePing), 0},
		{Version, 99, 0},
		{Version, byte(TypePing), 0, 1},
	} {
		if _, err := Decode(frame); err == nil {
			t.Fatalf("Decode(%x) error = nil", frame)
		}
	}
}

func BenchmarkEncodeSnapshotBinary(b *testing.B) {
	envelope, _ := NewEnvelope(TypeSnapshot, "", benchmarkSnapshot())
	b.ReportAllocs()
	for b.Loop() {
		if _, err := Encode(envelope); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEncodeSnapshotSonicJSON(b *testing.B) {
	snapshot := benchmarkSnapshot()
	b.ReportAllocs()
	for b.Loop() {
		if _, err := sonic.Marshal(snapshot); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecodeSnapshotBinary(b *testing.B) {
	envelope, _ := NewEnvelope(TypeSnapshot, "", benchmarkSnapshot())
	frame, err := Encode(envelope)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := Decode(frame); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecodeSnapshotSonicJSON(b *testing.B) {
	frame, err := sonic.Marshal(benchmarkSnapshot())
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		var snapshot SnapshotPayload
		if err := sonic.Unmarshal(frame, &snapshot); err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkSnapshot() SnapshotPayload {
	snapshot := SnapshotPayload{
		Tick: 4200, ServerTimeMs: 1_780_000_000_000, RemainingMs: 180_000,
		Team: SnapshotTeam{Level: 7, Experience: 42, ExperienceRequired: 77, TotalKills: 312, ProjectileCount: 4, PickupRadius: 228},
	}
	for index := range 4 {
		snapshot.Players = append(snapshot.Players, SnapshotPlayer{
			ID: "p_abcdefghijklmnop", DisplayName: "Player", X: 1500.25 + float64(index), Y: 900.5,
			VelocityX: 155.56, VelocityY: -155.56, MovementSpeed: 325.6, ArmorPercent: 0.6, Facing: "right", HP: 100, MaxHP: 100,
			Alive: true, LastProcessedInput: uint64(1000 + index), Kills: 78,
		})
	}
	for index := range 150 {
		snapshot.Monsters = append(snapshot.Monsters, SnapshotMonster{ID: uint64(index + 1), X: float64(500 + index*7), Y: float64(300 + index*3), HP: 40, MaxHP: 40})
	}
	for index := range 50 {
		snapshot.Pickups = append(snapshot.Pickups, SnapshotPickup{ID: uint64(index + 1), Kind: "experience", X: float64(600 + index*5), Y: float64(400 + index*2)})
	}
	return snapshot
}
