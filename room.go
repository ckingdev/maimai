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
	conn   connection
	data   *roomData
	config *RoomConfig
}

// NewRoom creates a new room with the given configurations.
func NewRoom(roomCfg *RoomConfig, conn connection) (*Room, error) {
	return &Room{conn, &roomData{0, make(map[string]time.Time)}, roomCfg}, nil
}
