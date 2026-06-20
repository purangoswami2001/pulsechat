package websocket

import "encoding/json"

type Event struct {
	Type    string          `json:"type"`
	RoomID  string          `json:"room_id"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type BroadcastPacket struct {
	RoomID        string
	Payload       []byte
	ExcludeClient interface{}
}
