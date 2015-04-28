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
	connect() error
	start(inbound chan *PacketEvent, outbound chan *PacketEvent)
	stop()
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

func (ws *WSSenderReceiver) connect() error {
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
		if err = ws.connect(); err != nil {
			return err
		}
		err = ws.conn.WriteJSON(msg)
		return err
	}
	return nil
}

func (ws *WSSenderReceiver) sender(outbound chan *PacketEvent) {
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
		if err = ws.connect(); err != nil {
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

func (ws *WSSenderReceiver) receivePacket(packetCh chan *PacketEvent) {
	packet, err := ws.receiveMessage()
	if err != nil {
		panic(err)
	}
	packetCh <- packet
}

func (ws *WSSenderReceiver) receiver(inbound chan *PacketEvent) {
	for {
		packetCh := make(chan *PacketEvent)
		go ws.receivePacket(packetCh)
		select {
		case packet := <-packetCh:
			inbound <- packet
		case <-ws.stopChan:
			return
		}
	}
}

func (ws *WSSenderReceiver) start(inbound chan *PacketEvent, outbound chan *PacketEvent) {
	ws.wg.Add(1)
	go func() {
		defer ws.wg.Done()
		ws.receiver(inbound)
	}()
	ws.wg.Add(1)
	go func() {
		defer ws.wg.Done()
		ws.sender(outbound)
	}()
}

func (ws *WSSenderReceiver) stop() {
	ws.stopChan <- empty{}
	ws.stopChan <- empty{}
	ws.wg.Wait()
}
