package maimai

import (
	"fmt"
	"testing"
	"time"
)

type TestHarness struct {
	outbound *(chan interface{})
	inbound  *(chan *PacketEvent)
}

func NewTestHarness() (*Room, *TestHarness) {
	roomCfg := &RoomConfig{"MaiMai", "", "test.db", "test.log"}
	room, err := NewRoom(roomCfg, "test")
	if err != nil {
		panic(err)
	}
	th := &TestHarness{&room.outbound, &room.inbound}
	return room, th
}

func (th *TestHarness) AssertReceivedSendText(text string) {
	msg := <-*th.outbound
	packet, ok := msg.(*PacketEvent)
	if !ok {
		panic("Could not assert message as *PacketEvent.")
	}
	if packet.Type != SendEventType {
		panic("Packet is not of type 'send'.")
	}
	payload, err := packet.Payload()
	if err != nil {
		panic("Could not extract packet payload.")
	}
	data, ok := payload.(SendEvent)
	if !ok {
		panic("Could not assert payload as SendEvent.")
	}
	if data.Content != text {
		panic(fmt.Sprintf("Message content does not match text. Expected '%s', got '%s'", text, data.Content))
	}
}

func (th *TestHarness) AssertReceivedNick() {
	msg := <-*th.outbound
	packet, ok := msg.(*PacketEvent)
	if !ok {
		panic("Could not assert message as *PacketEvent.")
	}
	if packet.Type != "nick" {
		panic("Packet is not of type 'nick'.")
	}
}

func TestConnect(t *testing.T) {
	room, _ := NewTestHarness()
	defer room.db.Close()
	if err := room.sr.Connect("test"); err != nil {
		t.Fatal("Could not connect to mock interface.")
	}
}

func TestRun(t *testing.T) {
	room, _ := NewTestHarness()
	defer room.db.Close()
	go room.Run()
	time.Sleep(time.Second * time.Duration(3))
}

// func TestSendText(t *testing.T) {
// 	room, th := NewTestHarness()
// 	defer room.db.Close()
// 	go room.Run()
// 	room.SendText("test text", "")
// 	th.AssertReceivedSendText("test text")
// }
