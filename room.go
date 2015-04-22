package maimai

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/boltdb/bolt"
)

type roomData struct {
	msgID int
	seen  map[string]time.Time
}

// RoomConfig stores configuration options specific to a Room.
type RoomConfig struct {
	Nick         string
	MsgPrefix    string
	DBPath       string
	ErrorLogPath string
}

// Room represents a connection to a euphoria room and associated data.
type Room struct {
	conn     connection
	data     *roomData
	config   *RoomConfig
	db       *bolt.DB
	handlers []Handler
}

// NewRoom creates a new room with the given configurations.
func NewRoom(roomCfg *RoomConfig, conn connection) (*Room, error) {
	log.Println("Creating/opening db...")
	db, err := bolt.Open(roomCfg.DBPath, 0666, nil)
	log.Println("Opened db.")
	if err != nil {
		return nil, err
	}
	handlers := make([]Handler, 0)
	// TODO : change this to read handler config from file
	handlers = append(handlers, PingEventHandler)
	handlers = append(handlers, PingCommandHandler)
	handlers = append(handlers, SeenCommandHandler)
	handlers = append(handlers, SeenRecordHandler)
	return &Room{conn, &roomData{0, make(map[string]time.Time)}, roomCfg, db, handlers}, nil
}

func (r *Room) Auth(password string) error {
	msg := map[string]interface{}{
		"type": "auth",
		"data": map[string]string{"type": "passcode",
		"passcode": password}}
	err := r.conn.sendJSON(msg)
	r.data.msgID++
	return err
}


// SendText sends a text message to the euphoria room.
func (r *Room) SendText(text string, parent string) error {
	msg := map[string]interface{}{
		"data": map[string]string{"content": r.config.MsgPrefix + text, "parent": parent},
		"type": "send", "id": strconv.Itoa(r.data.msgID)}
	err := r.conn.sendJSON(msg)
	r.data.msgID++
	return err
}

// SendPing sends a ping-reply, used in response to a ping-event.
func (r *Room) SendPing(time int64) error {
	msg := map[string]interface{}{"type": "ping-reply",
		"id": strconv.Itoa(r.data.msgID), "data": map[string]int64{
			"time": time}}
	err := r.conn.sendJSON(msg)
	r.data.msgID++
	return err
}

// SendNick sends a nick-event, setting the bot's nickname in the room.
func (r *Room) SendNick(nick string) error {
	msg := map[string]interface{}{
		"type": "nick",
		"data": map[string]string{"name": nick},
		"id":   strconv.Itoa(r.data.msgID)}
	err := r.conn.sendJSON(msg)
	r.data.msgID++
	return err
}

func (r *Room) storeSeen(user string, time int64) error {
	err := r.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("Seen"))
		if err != nil {
			return fmt.Errorf("Error creating bucket 'Seen': %s", err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	err = r.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Seen"))
		b.Put([]byte(user), []byte(strconv.FormatInt(time, 10)))
		return nil
	})
	return err
}

func (r *Room) retrieveSeen(user string) ([]byte, error) {
	err := r.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("Seen"))
		if err != nil {
			return fmt.Errorf("Error creating bucket 'Seen': %s", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	var t []byte
	err = r.db.View(func(tx *bolt.Tx) error {
		t = tx.Bucket([]byte("Seen")).Get([]byte(user))
		return nil
	})
	return t, err
}

// Run provides a method for setup and the main loop that the bot will run with handlers.
func (r *Room) Run() {
	if r.config.ErrorLogPath != "" {
		errorFile, err := os.OpenFile(r.config.ErrorLogPath,
			os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			panic(err)
		}
		defer errorFile.Close()
		log.SetOutput(errorFile)
	}
	if DEBUG {
		log.Println("DEBUG: Setting nick.")
	}
	errChan := make(chan error, 1)
	for {
		if DEBUG {
			log.Println("DEBUG: Handling packet.")
		}
		packet, err := r.conn.receivePacketWithRetries()
		if err != nil {
			panic(err)
		}
		var wg sync.WaitGroup
		for _, handler := range r.handlers {
			wg.Add(1)
			go func(h Handler) {
				defer wg.Done()
				h(r, packet, errChan)
			}(handler)
		}
		wg.Wait()
		select {
		case <-errChan:
			panic(err)
		default:
			continue
		}
	}
}
