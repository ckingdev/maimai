package maimai

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

type connection interface {
	connect() error
	connectWithRetries() error
	receivePacket() (*PacketEvent, error)
	receivePacketWithRetries() (*PacketEvent, error)
	sendJSON(msg interface{}) error
}

// ConnConfig stores the configuration for a Conn.
type ConnConfig struct {
	Room       string
	Retries    int
	RetrySleep time.Duration
}

// Conn stores a websocket connection to a room and its configuration.
type Conn struct {
	ws  *websocket.Conn
	cfg *ConnConfig
}

func (c *Conn) connect() error {
	tlsConn, err := tls.Dial("tcp", "euphoria.io:443", &tls.Config{})
	if err != nil {
		return err
	}
	roomURL, err := url.Parse("wss://euphoria.io/room/" + c.cfg.Room + "/ws")
	if err != nil {
		return err
	}
	wsConn, _, err := websocket.NewClient(tlsConn, roomURL, http.Header{}, 4096, 4096)
	if err != nil {
		return err
	}
	c.ws = wsConn
	return nil
}

func (c *Conn) connectWithRetries() error {
	err := c.connect()
	if err != nil {
		for i := 0; i < c.cfg.Retries; i++ {
			time.Sleep(c.cfg.RetrySleep)
			err = c.connect()
			if err == nil {
				break
			}
		}
	}
	return err
}

// NewConn returns a new websocket connection to a room.
//
// Parameters:
// connCfg: configuration for the new connection to use.
func NewConn(connCfg *ConnConfig) (*Conn, error) {
	conn := Conn{nil, connCfg}
	if err := conn.connectWithRetries(); err != nil {
		return nil, err
	}
	return &conn, nil
}

func (c *Conn) sendJSON(msg interface{}) error {

	if err := c.ws.WriteJSON(msg); err != nil {
		if err = c.connectWithRetries(); err != nil {
			return err
		}
		err := c.ws.WriteJSON(msg)
		return err
	}
	return nil
}

func (c *Conn) receivePacket() (*PacketEvent, error) {
	_, msg, err := c.ws.ReadMessage()
	var packet PacketEvent

	if err = json.Unmarshal(msg, &packet); err != nil {
		return &PacketEvent{}, err
	}
	return &packet, nil
}

func (c *Conn) receivePacketWithRetries() (*PacketEvent, error) {
	packet, err := c.receivePacket()
	if err != nil {
		for i := 0; i < c.cfg.Retries; i++ {
			packet, err = c.receivePacket()
			if err == nil {
				break
			}
		}
	}
	return packet, err
}

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
