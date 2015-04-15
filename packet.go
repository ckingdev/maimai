package maimai

import (
	"encoding/json"
	"errors"
)

// PacketType indicates the type of a packet's payload.
type PacketType string

// PacketEvent is the skeleton of a packet, its payload is composed of another type or types.
type PacketEvent struct {
	ID    string          `json:"id"`
	Type  PacketType      `json:"type"`
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}

// Message is a unit of data associated with a text message sent on the service.
type Message struct {
	ID              string `json:"id"`
	Parent          string `json:"parent"`
	PreviousEditID  string `json:"previous_edit_id,omitempty"`
	Time            int    `json:"time"`
	Sender          User   `json:"sender"`
	Content         string `json:"content"`
	EncryptionKeyID string `json:"encryption_key_id,omitempty"`
	Edited          int    `json:"edited,omitempty"`
	Deleted         int    `json:"deleted,omitempty"`
}

// PingEvent encodes the server's information on when this ping occurred and when the next will.
type PingEvent struct {
	Time int64 `json:"time"`
	Next int64 `json:"next"`
}

// User encodes the information about a user in the room. Name may be duplicated within a room
type User struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ServerID  string `json:"server_id"`
	ServerEra string `json:"server_era"`
}

// SendEvent is a packet type that contains a Message only.
type SendEvent Message

// ReplyEvent is a packet type that contains a Message only.
type ReplyEvent Message

// JoinEvent is a packet type that contains a User only.
type JoinEvent User

// PartEvent is a packet type that contains a User only.
type PartEvent User

// NickEvent encodes the packet type sent when a user changes or initially sets their nick.
type NickEvent struct {
	ID   string `json:"id"`
	From string `json:"from"`
	To   string `json:"to"`
}

// SnapShotEvent is a packet that encodes the backlog of messages and userlist sent on connect.
type SnapShotEvent struct {
	Version   string      `json:"version"`
	Log       []SendEvent `json:"log"`
	SessionID string      `json:"session_id"`
	Listing   []User      `json:"listing"`
}

// NickReplyEvent is a packet type that contains a NickEvent only.
type NickReplyEvent NickEvent

// These give named constants to the packet types.
const (
	PingType           = "ping-event"
	PingReplyReplyType = "ping-reply-reply"
	SendType           = "send-event"
	ReplyType          = "send-reply"
	SnapshotType       = "snapshot-event"
	JoinType           = "join-event"
	NickType           = "nick-event"
	PartType           = "part-event"
	NetworkType        = "network-event"
	NickReplyType      = "nick-reply"
	BounceType         = "bounce-event"
	AuthReplyType      = "auth-reply"
)

type networkEvent string
type pingReplyReplyEvent string
type bounceEvent string
type authReplyEvent string

// Payload unmarshals the packet payload into the proper Event type and returns it.
func (p *PacketEvent) Payload() (interface{}, error) {
	var payload interface{}
	switch p.Type {
	case PingType:
		payload = &PingEvent{}
	case SendType:
		payload = &SendEvent{}
	case ReplyType:
		payload = &ReplyEvent{}
	case SnapshotType:
		payload = &SnapShotEvent{}
	case JoinType:
		payload = &JoinEvent{}
	case NickType:
		payload = &NickEvent{}
	case PartType:
		payload = &PartEvent{}
	case NickReplyType:
		payload = &NickReplyEvent{}
	default:
		return p.Data, errors.New("Unexpected packet type.")
	}
	err := json.Unmarshal(p.Data, &payload)
	return payload, err
}
