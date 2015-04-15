package maimai

import (
	"time"
)

type roomData struct {
	msgID int
	seen  map[string]time.Time
}

type RoomConfig struct {
	Nick      string
	MsgPrefix string
}

type Room struct {
	conn   *Conn
	data   *roomData
	config *RoomConfig
}

func NewRoom(roomCfg *RoomConfig, connCfg *ConnConfig) (*Room, error) {
	conn, err := NewConn(connCfg)
	if err != nil {
		return nil, err
	}
	return &Room{conn, &roomData{0, make(map[string]time.Time)}, roomCfg}, nil
}
