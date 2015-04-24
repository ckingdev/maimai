package maimai

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

type SenderReceiver interface {
	Connect(room string) error
	Sender(outbound chan interface{})
	Receiver(inbound chan *PacketEvent)
	Room() string
}

type WSSenderReceiver struct {
	conn *websocket.Conn
	room string
}

func (ws *WSSenderReceiver) connectOnce(room string) error {
	tlsConn, err := tls.Dial("tcp", "euphoria.io:443", &tls.Config{})
	if err != nil {
		return err
	}
	roomURL, err := url.Parse(fmt.Sprintf("wss://euphoria.io/room/%s/ws", room))
	if err != nil {
		return err
	}
	wsConn, _, err := websocket.NewClient(tlsConn, roomURL, http.Header{}, 4096, 4096)
	if err != nil {
		return err
	}
	ws.conn = wsConn
	return nil
}

func (ws *WSSenderReceiver) Connect(room string) error {
	if err := ws.connectOnce(room); err != nil {
		for i := 0; i < 5; i++ {
			time.Sleep(time.Duration(500) * time.Millisecond)
			err = ws.connectOnce(room)
			if err != nil {
				break
			}
		}
		return err
	}
	return nil
}

func (ws *WSSenderReceiver) sendJSON(msg interface{}) error {
	if err := ws.conn.WriteJSON(msg); err != nil {
		if err = ws.Connect(ws.room); err != nil {
			return err
		}
		err = ws.conn.WriteJSON(msg)
		return err
	}
	return nil
}

func (ws *WSSenderReceiver) Sender(outbound chan interface{}) {
	for {
		msg := <-outbound
		if err := ws.sendJSON(msg); err != nil {
			panic(err)
		}
	}
}

func (ws *WSSenderReceiver) receiveMessage() (*PacketEvent, error) {
	_, msg, err := ws.conn.ReadMessage()
	if err != nil {
		if err = ws.Connect(ws.room); err != nil {
			return &PacketEvent{}, err
		}
		_, msg, err = ws.conn.ReadMessage()
		if err != nil {
			return &PacketEvent{}, err
		}
	}
	var packet PacketEvent
	if err = json.Unmarshal(msg, &packet); err != nil {
		return &PacketEvent{}, fmt.Errorf("Error unmarshalling packet: %s", msg)
	}
	return &packet, nil
}

func (ws *WSSenderReceiver) Receiver(inbound chan *PacketEvent) {
	for {
		packet, err := ws.receiveMessage()
		if err != nil {
			panic(err)
		}
		inbound <- packet
	}
}

func (ws *WSSenderReceiver) Room() string {
	return ws.room
}

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
