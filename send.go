package maimai

import (
	"strconv"
)

func (c *Conn) sendJSON(msg interface{}) error {
	err := c.ws.WriteJSON(msg)
	if err != nil {
		err = c.connectWithRetries()
		if err != nil {
			return err
		}
		err = c.ws.WriteJSON(msg)
	}
	return err
}

func (r *Room) SendText(text string, parent string) error {
	msg := map[string]interface{}{
		"data": map[string]string{"content": r.config.MsgPrefix + text, "parent": parent},
		"type": "send", "id": strconv.Itoa(r.data.msgID)}
	err := r.conn.sendJSON(msg)
	r.data.msgID++
	return err
}

func (r *Room) SendPing(time int64) error {
	msg := map[string]interface{}{"type": "ping-reply",
		"id": strconv.Itoa(r.data.msgID), "data": map[string]int64{
			"time": time}}
	err := r.conn.sendJSON(msg)
	r.data.msgID++
	return err
}

func (r *Room) SendNick(nick string) error {
	msg := map[string]interface{}{
		"type": "nick",
		"data": map[string]string{"name": nick},
		"id":   strconv.Itoa(r.data.msgID)}
	err := r.conn.sendJSON(msg)
	r.data.msgID++
	return err
}
