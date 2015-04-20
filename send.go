package maimai

import (
	"strconv"
)

func (c *Conn) sendJSON(msg interface{}) error {

	if err := c.ws.WriteJSON(msg); err != nil {
		if err = c.connectWithRetries(); err != nil {
			return err
		}
		err := c.ws.WriteJSON(msg)
		return err
	}
	return nil
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
