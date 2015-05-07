package maimai

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Sirupsen/logrus"
)

type MockSenderReceiver struct {
	outbound chan *PacketEvent
	inbound  chan *PacketEvent
	stopFlag bool
	room     string
	wg       sync.WaitGroup
}

func NewMockSR(room string) *MockSenderReceiver {
	outbound := make(chan *PacketEvent, 4)
	inbound := make(chan *PacketEvent, 4)
	return &MockSenderReceiver{outbound, inbound, false, room, sync.WaitGroup{}}
}

func (m *MockSenderReceiver) connect() error {
	return nil
}

func (m *MockSenderReceiver) sender(outbound chan *PacketEvent) {
	for {
		if m.stopFlag {
			return
		}
		msg := <-outbound
		m.outbound <- msg
	}
}

func (m *MockSenderReceiver) receiver(inbound chan *PacketEvent) {
	for {
		if m.stopFlag {
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

func (m *MockSenderReceiver) start(inbound chan *PacketEvent, outbound chan *PacketEvent) {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.receiver(inbound)
	}()
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.sender(outbound)
	}()
}

func (m *MockSenderReceiver) stop() {
	m.stopFlag = true
}

type TestHarness struct {
	outbound *chan *PacketEvent
	inbound  *chan *PacketEvent
	t        *testing.T
}

func NewTestHarness(t *testing.T) (*Room, *TestHarness) {
	roomCfg := &RoomConfig{"MaiMai", "", "test.db", "test.log", true, true}
	mockSR := NewMockSR("test")
	room, err := NewRoom(roomCfg, "test", mockSR, logrus.New())
	if err != nil {
		panic(err)
	}
	th := &TestHarness{&mockSR.outbound, &mockSR.inbound, t}
	return room, th
}

func (th *TestHarness) AssertReceivedSendText(text string) {
	packet := <-*th.outbound
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

func (th *TestHarness) AssertReceivedSendPrefix(prefix string) {
	packet := <-*th.outbound
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
	if !strings.HasPrefix(data.Content, prefix) {
		th.t.Fatalf("Message beginning does not match prefix. Expected '%s', got '%s'", prefix, data.Content)
	}
}

func (th *TestHarness) AssertReceivedNick() {
	packet := <-*th.outbound
	if packet.Type != "nick" {
		th.t.Fatal("Packet is not of type 'nick'.")
	}
}

func (th *TestHarness) AssertReceivedAuth() {
	packet := <-*th.outbound
	if packet.Type != "auth" {
		th.t.Fatalf("Incorrect packet type. Expected 'auth', got '%s'.", packet.Type)
	}
	payload, err := packet.Payload()
	if err != nil {
		th.t.Fatalf("Could not extract packet payload. Error: %s", err)
	}
	data, ok := payload.(*AuthCommand)
	if !ok {
		th.t.Fatal("Could not assert payload as *AuthCommand.")
	}
	if data.Passcode != "test" {
		th.t.Fatal("Incorrect passcode.")
	}
	if data.Type != "passcode" {
		th.t.Fatal("Incorrect auth type.")
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

func (th *TestHarness) SendPingEvent() {
	payload, _ := json.Marshal(PingEvent{
		Time: time.Now().Unix(),
		Next: time.Now().Unix() + 30})
	msg := PacketEvent{
		Type: PingEventType,
		Data: payload}
	*th.inbound <- &msg
}

func (th *TestHarness) SendNickEvent(from string, to string) {
	payload, _ := json.Marshal(NickEvent{
		From: from,
		To:   to})
	msg := PacketEvent{
		Type: NickEventType,
		Data: payload}
	*th.inbound <- &msg
}

func (th *TestHarness) SendPresenceEvent(ptype PacketType, name string) {
	payload, _ := json.Marshal(PresenceEvent{
		User: &User{Name: name}})
	msg := PacketEvent{
		Type: ptype,
		Data: payload}
	*th.inbound <- &msg
}

func TestConnect(t *testing.T) {
	room, _ := NewTestHarness(t)
	defer room.db.Close()
	if err := room.sr.connect(); err != nil {
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
	defer room.Stop()
	go room.Run()
	room.SendText("test text", "")
	th.AssertReceivedSendText("test text")
}

func TestPingCommand(t *testing.T) {
	room, th := NewTestHarness(t)
	defer room.db.Close()
	defer room.Stop()
	go room.Run()
	th.SendSendEvent("!ping", "", "test")
	th.AssertReceivedSendText("pong!")
}

func TestScritchCommand(t *testing.T) {
	room, th := NewTestHarness(t)
	defer room.db.Close()
	go room.Run()
	th.SendSendEvent("!scritch", "", "test")
	th.AssertReceivedSendText("/me bruxes")
	defer room.Stop()
}

func TestSeenCommand(t *testing.T) {
	room, th := NewTestHarness(t)
	defer room.db.Close()
	go room.Run()
	th.SendSendEvent("!seen @xyz", "", "test")
	th.AssertReceivedSendText("User has not been seen yet.")
	th.SendSendEvent("!seen @test", "", "test")
	th.AssertReceivedSendText("Seen 0 hours ago.")
	defer room.Stop()
}

func TestUptimeCommand(t *testing.T) {
	room, th := NewTestHarness(t)
	defer room.db.Close()
	go room.Run()
	th.SendSendEvent("!uptime", "", "test")
	th.AssertReceivedSendPrefix("This bot has been up for")
	defer room.Stop()
}

func TestLinkTitle(t *testing.T) {
	room, th := NewTestHarness(t)
	defer room.db.Close()
	go room.Run()
	th.SendSendEvent("google.com", "", "test")
	th.AssertReceivedSendText("Link title: Google")
	// Does not exist
	th.SendSendEvent("foo.bar", "", "test")
	select {
	case <-*th.outbound:
		panic("Unexpected packet.")
	case <-time.After(time.Duration(300) * time.Millisecond):
		break
	}
	// 404
	th.SendSendEvent("http://www.google.com/microsoft", "", "test")
	select {
	case <-*th.outbound:
		panic("Unexpected packet.")
	case <-time.After(time.Duration(300) * time.Millisecond):
		break
	}
	// No <title>
	th.SendSendEvent("https://www.gutenberg.org/cache/epub/48797/pg48797.txt", "", "test")
	select {
	case <-*th.outbound:
		panic("Unexpected packet.")
	case <-time.After(time.Duration(300) * time.Millisecond):
		break
	}
	// Imgur-only title
	th.SendSendEvent("https://imgur.com/aFga8B9", "", "test")
	select {
	case <-*th.outbound:
		panic("Unexpected packet.")
	case <-time.After(time.Duration(300) * time.Millisecond):
		break
	}
	defer room.Stop()
}

func TestPingReply(t *testing.T) {
	room, th := NewTestHarness(t)
	go room.Run()
	defer room.db.Close()
	defer room.Stop()
	th.SendPingEvent()
	packet := <-*th.outbound
	if packet.Type != PingReplyType {
		t.Fatalf("Incorrect packet type. Expected 'ping-reply', got '%s'", packet.Type)
	}
	_, err := packet.Payload()
	if err != nil {
		t.Fatal("Could not extract payload.")
	}
}

func TestWS(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode.")
	}
	roomCfg := &RoomConfig{"MaiMai", "", "test.db", "test.log", true, true}
	room, err := NewRoom(roomCfg, "test", NewWSSenderReceiver("test", logrus.New()), logrus.New())
	if err != nil {
		panic(err)
	}
	defer room.db.Close()
	defer room.Stop()
	room.SendNick(roomCfg.Nick)
	go room.Run()
	time.Sleep(time.Duration(60) * time.Second)
}

func TestNickChange(t *testing.T) {
	room, th := NewTestHarness(t)
	defer room.db.Close()
	go room.Run()
	th.SendNickEvent("test1", "test2")
	th.AssertReceivedSendText("< test1 is now known as test2. >")
	defer room.Stop()
}

func TestJoin(t *testing.T) {
	room, th := NewTestHarness(t)
	defer room.db.Close()
	go room.Run()
	th.SendNickEvent("", "test1")
	th.AssertReceivedSendText("< test1 joined the room. >")
	th.SendPresenceEvent("join-event", "test2")
	th.AssertReceivedSendText("< test2 joined the room. >")
	defer room.Stop()
}

func TestPart(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	room, th := NewTestHarness(t)
	defer room.db.Close()
	defer room.Stop()
	go room.Run()
	th.SendPresenceEvent("part-event", "test1")
	select {
	case msg := <-*th.outbound:
		if msg.Type != SendType {
			t.Fatalf("Incorrect packet type. Expected 'send', got '%s'.", msg.Type)
		}
	case <-time.After(time.Duration(65) * time.Second):
		t.Fatal("Timeout: expecting send packet.")
	}
}

func TestBadWS(t *testing.T) {
	roomCfg := &RoomConfig{"MaiMai", "", "test.db", "test.log", true, true}
	room, err := NewRoom(roomCfg, "test/bad/room", NewWSSenderReceiver("test/bad/room", logrus.New()), logrus.New())
	if err != nil {
		panic(err)
	}
	defer room.db.Close()
	// defer room.Stop()
	go room.Run()
}

func TestSendAuth(t *testing.T) {
	room, th := NewTestHarness(t)
	defer room.db.Close()
	defer room.Stop()
	go room.Run()
	room.SendAuth("test")
	th.AssertReceivedAuth()
}
