package maimai

import (
	"fmt"
	"log"
	"os"
	"strconv"
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
	data     *roomData
	config   *RoomConfig
	db       *bolt.DB
	handlers []Handler
	uptime   time.Time
	inbound  chan *PacketEvent
	outbound chan interface{}
	errChan  chan error
	sr       SenderReceiver
}

// NewRoom creates a new room with the given configurations.
func NewRoom(roomCfg *RoomConfig, room string) (*Room, error) {
	log.Println("Creating/opening db...")
	db, err := bolt.Open(roomCfg.DBPath, 0666, nil)
	log.Println("Opened db.")
	if err != nil {
		return nil, err
	}
	var handlers []Handler
	// TODO : change this to read handler config from file
	handlers = append(handlers, PingEventHandler)
	handlers = append(handlers, PingCommandHandler)
	handlers = append(handlers, SeenCommandHandler)
	handlers = append(handlers, SeenRecordHandler)
	handlers = append(handlers, LinkTitleHandler)
	handlers = append(handlers, UptimeCommandHandler)
	handlers = append(handlers, ScritchCommandHandler)
	inbound := make(chan *PacketEvent, 4)
	outbound := make(chan interface{}, 4)
	errChan := make(chan error)
	sr := &WSSenderReceiver{room: room}
	return &Room{&roomData{0, make(map[string]time.Time)}, roomCfg, db, handlers, time.Now(), inbound, outbound, errChan, sr}, nil
}

// Auth sends an authentication packet with the given password.
func (r *Room) Auth(password string) {
	msg := map[string]interface{}{
		"type": "auth",
		"data": map[string]string{"type": "passcode",
			"passcode": password}}
	r.outbound <- msg
	r.data.msgID++
}

// SendText sends a text message to the euphoria room.
func (r *Room) SendText(text string, parent string) {
	msg := map[string]interface{}{
		"data": map[string]string{"content": r.config.MsgPrefix + text, "parent": parent},
		"type": "send", "id": strconv.Itoa(r.data.msgID)}
	r.outbound <- msg
	r.data.msgID++
}

// SendPing sends a ping-reply, used in response to a ping-event.
func (r *Room) SendPing(time int64) {
	msg := map[string]interface{}{
		"type": "ping-reply",
		"id":   strconv.Itoa(r.data.msgID),
		"data": map[string]int64{"time": time}}
	r.outbound <- msg
	r.data.msgID++
}

// SendNick sends a nick-event, setting the bot's nickname in the room.
func (r *Room) SendNick(nick string) {
	msg := map[string]interface{}{
		"type": "nick",
		"data": map[string]string{"name": nick},
		"id":   strconv.Itoa(r.data.msgID)}
	r.outbound <- msg
	r.data.msgID++
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

func (r *Room) dispatcher() {
	var fanout [](chan PacketEvent)
	for i, h := range r.handlers {
		fanout = append(fanout, make(chan PacketEvent, 4))
		go h(r, fanout[i])
	}
	for {
		inboundMsg := <-r.inbound
		for _, channel := range fanout {
			channel <- *inboundMsg
		}

	}
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
	if err := r.sr.Connect(r.sr.Room()); err != nil {
		panic(err)
	}
	go r.dispatcher()
	go r.sr.Receiver(r.inbound)
	go r.sr.Sender(r.outbound)
	<-r.errChan
}
