package maimai

import (
	"strconv"
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

// SendText sends a text message to the euphoria room.
func (r *Room) SendText(text string, parent string) error {
	msg := map[string]interface{}{
		"data": map[string]string{"content": r.config.MsgPrefix + text, "parent": parent},
		"type": "send", "id": strconv.Itoa(r.data.msgID)}
	err := r.conn.sendJSON(msg)
	r.data.msgID++
	return err
}

// SendPing sends a ping-reply, used in response to a ping-event.
func (r *Room) SendPing(time int64) error {
	msg := map[string]interface{}{"type": "ping-reply",
		"id": strconv.Itoa(r.data.msgID), "data": map[string]int64{
			"time": time}}
	err := r.conn.sendJSON(msg)
	r.data.msgID++
	return err
}

// SendNick sends a nick-event, setting the bot's nickname in the room.
func (r *Room) SendNick(nick string) error {
	msg := map[string]interface{}{
		"type": "nick",
		"data": map[string]string{"name": nick},
		"id":   strconv.Itoa(r.data.msgID)}
	err := r.conn.sendJSON(msg)
	r.data.msgID++
	return err
}
