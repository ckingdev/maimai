package maimai

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

type MockSenderReceiver struct {
	outbound chan *interface{}
	inbound  chan *PacketEvent
	fail     bool
	room     string
}

func NewMockSR(room string) *MockSenderReceiver {
	outbound := make(chan *interface{}, 4)
	inbound := make(chan *PacketEvent, 4)
	return &MockSenderReceiver{outbound, inbound, true, room}
}

func (m *MockSenderReceiver) Connect(room string) error {
	m.room = room
	if m.fail {
		return errors.New("Mock failed connect")
		m.fail = false
	}
	m.fail = true
	return nil
}

func (m *MockSenderReceiver) Sender(outbound chan *interface{}) {
	for {
		msg := <-outbound
		m.outbound <- msg
	}
}

func (m *MockSenderReceiver) Receiver(inbound chan *PacketEvent) {
	for {
		msg := <-m.inbound
		inbound <- msg
	}
}

type TestHarness struct {
	outbound *(chan interface{})
	inbound  *(chan *PacketEvent)
	t        *testing.T
}

func NewTestHarness(t *testing.T) (*Room, *TestHarness) {
	roomCfg := &RoomConfig{"MaiMai", "", "test.db", "test.log"}
	room, err := NewRoom(roomCfg, "test")
	if err != nil {
		panic(err)
	}
	th := &TestHarness{&room.outbound, &room.inbound, t}
	return room, th
}

func (th *TestHarness) AssertReceivedSendText(text string) {
	msg := <-*th.outbound
	packet, ok := msg.(PacketEvent)
	if !ok {
		th.t.Fatal("Could not assert message as PacketEvent.")
	}
	if packet.Type != SendType {
		th.t.Fatalf("Packet is not of type 'send'. Got %s", packet.Type)
	}
	payload, err := packet.Payload()
	if err != nil {
		th.t.Fatalf("Could not extract packet payload. Error: %s", err)
	}
	data, ok := payload.(*SendCommand)
	if !ok {
		th.t.Fatal("Could not assert payload as *SendCommand.")
	}
	if data.Content != text {
		th.t.Fatalf("Message content does not match text. Expected '%s', got '%s'", text, data.Content)
	}
}

func (th *TestHarness) AssertReceivedNick() {
	msg := <-*th.outbound
	packet, ok := msg.(*PacketEvent)
	if !ok {
		th.t.Fatal("Could not assert message as *PacketEvent.")
	}
	if packet.Type != "nick" {
		th.t.Fatal("Packet is not of type 'nick'.")
	}
}

func (th *TestHarness) SendSendEvent(text string, parent string, sender string) {
	payload, _ := json.Marshal(Message{
		Content: text,
		Parent:  parent,
		Sender:  User{Name: sender}})
	msg := PacketEvent{
		Type: SendEventType,
		Data: payload}
	*th.inbound <- &msg
}

func TestConnect(t *testing.T) {
	room, _ := NewTestHarness(t)
	defer room.db.Close()
	if err := room.sr.Connect("test"); err != nil {
		t.Fatal("Could not connect to mock interface.")
	}
	time.Sleep(time.Second)
}

func TestRun(t *testing.T) {
	room, _ := NewTestHarness(t)
	defer room.db.Close()
	go room.Run()
	time.Sleep(time.Second * time.Duration(3))
}

func TestSendText(t *testing.T) {
	room, th := NewTestHarness(t)
	defer room.db.Close()
	go room.Run()
	room.SendText("test text", "")
	th.AssertReceivedSendText("test text")
	time.Sleep(time.Second)
}

func TestPingCommand(t *testing.T) {
	room, th := NewTestHarness(t)
	defer room.db.Close()
	go room.Run()
	th.SendSendEvent("!ping", "", "test")
	th.AssertReceivedSendText("pong!")
	time.Sleep(time.Second)
}

func TestScritchCommand(t *testing.T) {
	room, th := NewTestHarness(t)
	defer room.db.Close()
	go room.Run()
	th.SendSendEvent("!scritch", "", "test")
	th.AssertReceivedSendText("/me bruxes")
	time.Sleep(time.Second)
}
