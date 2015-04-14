package maimai

import (
	"encoding/json"
)

func (c *Conn) receivePacket() (*PacketEvent, error) {
	_, msg, err := c.ws.ReadMessage()
	var packet PacketEvent
	err = json.Unmarshal(msg, &packet)
	if err != nil {
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

func (c *Conn) receivePayloadAndType() (PacketType, *interface{}, error) {
	packet, err := c.receivePacketWithRetries()
	if err != nil {
		return "", nil, err
	}
	payload, err := packet.Payload()
	if err != nil {
		return "", nil, err
	}
	return packet.Type, &payload, nil
}
