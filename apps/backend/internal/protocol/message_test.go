package protocol

import (
	"encoding/json"
	"testing"
)

func TestDecodePayloadIsStrict(t *testing.T) {
	envelope := Envelope{Payload: json.RawMessage(`{"displayName":"Umar","reconnectToken":null,"unexpected":true}`)}
	var payload JoinRoomPayload
	if err := envelope.DecodePayload(&payload); err == nil {
		t.Fatal("DecodePayload() error = nil, want unknown field error")
	}
}

func TestNewEnvelopeUsesProtocolVersion(t *testing.T) {
	envelope, err := NewEnvelope(TypePing, "request-1", struct{}{})
	if err != nil {
		t.Fatalf("NewEnvelope() error = %v", err)
	}
	if envelope.Version != Version || envelope.Type != TypePing || envelope.RequestID != "request-1" {
		t.Fatalf("unexpected envelope: %#v", envelope)
	}
}
