package maimai

import (
	"encoding/json"
	"testing"
	"time"
)

type MockSenderReceiver struct {
	outbound chan *interface{}
	inbound  chan *PacketEvent
	stop     bool
	room     string
}

func NewMockSR(room string) *MockSenderReceiver {
	outbound := make(chan *interface{}, 4)
	inbound := make(chan *PacketEvent, 4)
	return &MockSenderReceiver{outbound, inbound, false, room}
}

func (m *MockSenderReceiver) Connect(room string) error {
	m.room = room
	return nil
}

func (m *MockSenderReceiver) Sender(outbound chan interface{}) {
	for {
		if m.stop {
			return
		}
		msg := <-outbound
		m.outbound <- &msg
	}
}

func (m *MockSenderReceiver) Receiver(inbound chan *PacketEvent) {
	for {
		if m.stop {
			return
		}
		select {
		case msg := <-m.inbound:
			inbound <- msg
		case <-time.After(time.Duration(50) * time.Millisecond):
			continue
		}
	}
}

func (m *MockSenderReceiver) Room() string {
	return m.room
}

func (m *MockSenderReceiver) Stop() {
	m.stop = true
}

type TestHarness struct {
	outbound *chan *interface{}
	inbound  *chan *PacketEvent
	t        *testing.T
}

func NewTestHarness(t *testing.T) (*Room, *TestHarness) {
	roomCfg := &RoomConfig{"MaiMai", "", "test.db", "test.log"}
	mockSR := NewMockSR("test")
	room, err := NewRoom(roomCfg, "test", mockSR)
	if err != nil {
		panic(err)
	}
	th := &TestHarness{&mockSR.outbound, &mockSR.inbound, t}
	return room, th
}

func (th *TestHarness) AssertReceivedSendText(text string) {
	msg := <-*th.outbound
	packet, ok := (*msg).(PacketEvent)
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
	packet, ok := (*msg).(*PacketEvent)
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
}

func TestRun(t *testing.T) {
	room, _ := NewTestHarness(t)
	defer room.db.Close()
	go room.Run()
	room.Stop()
}

func TestSendText(t *testing.T) {
	room, th := NewTestHarness(t)
	defer room.db.Close()
	go room.Run()
	room.SendText("test text", "")
	th.AssertReceivedSendText("test text")
	room.Stop()
}

func TestPingCommand(t *testing.T) {
	room, th := NewTestHarness(t)
	defer room.db.Close()
	go room.Run()
	th.SendSendEvent("!ping", "", "test")
	th.AssertReceivedSendText("pong!")
	room.Stop()
}

func TestScritchCommand(t *testing.T) {
	time.Sleep(time.Duration(1000) * time.Millisecond)
	room, th := NewTestHarness(t)
	defer room.db.Close()
	go room.Run()
	th.SendSendEvent("!scritch", "", "test")
	th.AssertReceivedSendText("/me bruxes")
	room.Stop()
}

func TestSeenCommand(t *testing.T) {
	room, th := NewTestHarness(t)
	defer room.db.Close()
	go room.Run()
	th.SendSendEvent("!seen @xyz", "", "test")
	th.AssertReceivedSendText("User has not been seen yet.")
	th.SendSendEvent("!seen @test", "", "test")
	th.AssertReceivedSendText("Seen 0 hours and 0 minutes ago.\n")
	room.Stop()
}
