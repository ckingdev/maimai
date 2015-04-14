package maimai

import (
	"encoding/json"
	"errors"
)

type PacketType string

type PacketEvent struct {
	ID    string          `json:"id"`
	Type  PacketType      `json:"type"`
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}

type PingEvent struct {
	Time int64 `json:"time"`
	Next int64 `json:"next"`
}

type User struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ServerID  string `json:"server_id"`
	ServerEra string `json:"server_era"`
}

type SendEvent struct {
	ID      string `json:"id"`
	Parent  string `json:"parent"`
	Time    int    `json:"time"`
	Sender  User   `json:"sender"`
	Content string `json:"content"`
}

type ReplyEvent struct {
	ID      string `json:"id"`
	Parent  string `json:"parent"`
	Time    int    `json:"time"`
	Sender  User   `json:"sender"`
	Content string `json:"content"`
}

type JoinEvent User

type PartEvent User

type NickEvent struct {
	ID   string `json:"id"`
	From string `json:"from"`
	To   string `json:"to"`
}

type SnapShotEvent struct {
	Version   string      `json:"version"`
	Log       []SendEvent `json:"log"`
	SessionID string      `json:"session_id"`
	Listing   []User      `json:"listing"`
}

type NickReplyEvent struct {
	ID   string `json:"id"`
	From string `json:"from"`
	To   string `json:"to"`
}

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
