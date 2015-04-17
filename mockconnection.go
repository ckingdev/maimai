package maimai

//type connection interface {
//	connect() error
//	connectWithRetries() error
//	receivePacket() (*PacketEvent, error)
//	receivePacketWithRetries() (*PacketEvent, error)
//	sendJSON(msg interface{}) error
//}

type mockConnection struct {
	inbound  *(chan PacketEvent)
	outbound *(chan interface{})
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
	*c.outbound <- msg
	return nil
}
