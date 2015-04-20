package maimai

import (
	"encoding/json"
)

func (c *Conn) receivePacket() (*PacketEvent, error) {
	_, msg, err := c.ws.ReadMessage()
	var packet PacketEvent

	if err = json.Unmarshal(msg, &packet); err != nil {
		return &PacketEvent{}, err
	}
	return &packet, nil
}

func (c *Conn) receivePacketWithRetries() (*PacketEvent, error) {
	packet, err := c.receivePacket()
	if err != nil {
		for i := 0; i < c.cfg.Retries; i++ {
			packet, err = c.receivePacket()
			if err == nil {
				break
			}
		}
	}
	return packet, err
}
