package maimai

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type SenderReceiver interface {
	Connect() error
	Start(inbound chan *PacketEvent, outbound chan *PacketEvent)
	GetRoom() string
	Stop()
}

type WSSenderReceiver struct {
	conn     *websocket.Conn
	Room     string
	stopChan chan empty
	wg       sync.WaitGroup
}

func NewWSSenderReceiver(room string) *WSSenderReceiver {
	return &WSSenderReceiver{
		Room:     room,
		stopChan: make(chan empty, 2)}
}

func (ws *WSSenderReceiver) connectOnce() error {
	tlsConn, err := tls.Dial("tcp", "euphoria.io:443", &tls.Config{})
	if err != nil {
		return err
	}
	roomURL, err := url.Parse(fmt.Sprintf("wss://euphoria.io/room/%s/ws", ws.Room))
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

func (ws *WSSenderReceiver) Connect() error {
	if err := ws.connectOnce(); err != nil {
		for i := 0; i < 5; i++ {
			time.Sleep(time.Duration(500) * time.Millisecond)
			err = ws.connectOnce()
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
		if err = ws.Connect(); err != nil {
			return err
		}
		err = ws.conn.WriteJSON(msg)
		return err
	}
	return nil
}

func (ws *WSSenderReceiver) Sender(outbound chan *PacketEvent) {
	for {
		select {
		case msg := <-outbound:
			if err := ws.sendJSON(msg); err != nil {
				panic(err)
			}
		case <-ws.stopChan:
			return
		}
	}
}

func (ws *WSSenderReceiver) receiveMessage() (*PacketEvent, error) {
	_, msg, err := ws.conn.ReadMessage()
	if err != nil {
		if err = ws.Connect(); err != nil {
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

func (ws *WSSenderReceiver) ReceivePacket(packetCh chan *PacketEvent) {
	packet, err := ws.receiveMessage()
	if err != nil {
		panic(err)
	}
	packetCh <- packet
}

func (ws *WSSenderReceiver) Receiver(inbound chan *PacketEvent) {
	for {
		packetCh := make(chan *PacketEvent)
		go ws.ReceivePacket(packetCh)
		select {
		case packet := <-packetCh:
			inbound <- packet
		case <-ws.stopChan:
			return
		}
	}
}

func (ws *WSSenderReceiver) GetRoom() string {
	return ws.Room
}

func (ws *WSSenderReceiver) Start(inbound chan *PacketEvent, outbound chan *PacketEvent) {
	ws.wg.Add(1)
	go func() {
		defer ws.wg.Done()
		ws.Receiver(inbound)
	}()
	ws.wg.Add(1)
	go func() {
		defer ws.wg.Done()
		ws.Sender(outbound)
	}()
}

func (ws *WSSenderReceiver) Stop() {
	ws.stopChan <- empty{}
	ws.stopChan <- empty{}
	ws.wg.Wait()
}
