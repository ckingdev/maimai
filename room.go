package maimai

import (
	"time"
)

type roomData struct {
	msgID int
	seen  map[string]time.Time
}

// RoomConfig stores configuration options specific to a Room.
type RoomConfig struct {
	Nick      string
	MsgPrefix string
}

// Room represents a connection to a euphoria room and associated data.
type Room struct {
	conn   *Conn
	data   *roomData
	config *RoomConfig
}

// NewRoom creates a new room with the given configurations.
func NewRoom(roomCfg *RoomConfig, connCfg *ConnConfig) (*Room, error) {
	conn, err := NewConn(connCfg)
	if err != nil {
		return nil, err
	}
	return &Room{conn, &roomData{0, make(map[string]time.Time)}, roomCfg}, nil
}
