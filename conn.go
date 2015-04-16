package maimai

import (
	"crypto/tls"
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
	err := conn.connectWithRetries()
	if err != nil {
		return nil, err
	}
	return &conn, nil
}
