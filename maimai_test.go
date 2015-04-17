package maimai

import (
	"testing"
)

func TestRoomReceive(t *testing.T) {
	inbound := make(chan PacketEvent, 1)
	outbound := make(chan interface{}, 1)
	mockConn := mockConnection{&inbound, &outbound}
	room, err := NewRoom(&RoomConfig{"MaiMai", ""}, mockConn)
	if err != nil {
		panic(err)
	}
	inbound <- PacketEvent{}

	_, err = room.conn.receivePacket()
	if err != nil {
		panic(err)
	}
	inbound <- PacketEvent{}
	_, err = room.conn.receivePacketWithRetries()
	if err != nil {
		panic(err)
	}
}

func TestRoomSend(t *testing.T) {
	inbound := make(chan PacketEvent, 1)
	outbound := make(chan interface{}, 1)
	mockConn := mockConnection{&inbound, &outbound}
	room, err := NewRoom(&RoomConfig{"MaiMai", ""}, mockConn)
	if err != nil {
		panic(err)
	}
	err = room.SendText("test text", "parent")
	if err != nil {
		panic(err)
	}
}
