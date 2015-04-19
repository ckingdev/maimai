package maimai

import (
	"encoding/json"
)

//type connection interface {
//	connect() error
//	connectWithRetries() error
//	receivePacket() (*PacketEvent, error)
//	receivePacketWithRetries() (*PacketEvent, error)
//	sendJSON(msg interface{}) error
//}

type mockConnection struct {
	inbound  *(chan PacketEvent)
	outbound *(chan []byte)
}

func (c mockConnection) receivePacket() (*PacketEvent, error) {
	packet := <-*(c.inbound)
	return &packet, nil
}

func (c mockConnection) receivePacketWithRetries() (*PacketEvent, error) {
	return c.receivePacket()
}

func (c mockConnection) connect() error {
	return nil
}

func (c mockConnection) connectWithRetries() error {
	return nil
}

func (c mockConnection) sendJSON(msg interface{}) error {
	marshalled, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	*c.outbound <- marshalled
	return nil
}
