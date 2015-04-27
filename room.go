package maimai

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/boltdb/bolt"
)

type empty struct{}

type roomData struct {
	msgID       int
	seen        map[string]time.Time
	userLeaving map[string]empty
}

// RoomConfig stores configuration options specific to a Room.
type RoomConfig struct {
	Nick         string
	MsgPrefix    string
	DBPath       string
	ErrorLogPath string
	Join         bool
}

// Room represents a connection to a euphoria room and associated data.
type Room struct {
	data     *roomData
	config   *RoomConfig
	db       *bolt.DB
	handlers []Handler
	uptime   time.Time
	inbound  chan *PacketEvent
	outbound chan *PacketEvent
	errChan  chan error
	sr       SenderReceiver
	cmdChan  chan string
}

// NewRoom creates a new room with the given configurations.
func NewRoom(roomCfg *RoomConfig, room string, sr SenderReceiver) (*Room, error) {
	db, err := bolt.Open(roomCfg.DBPath, 0666, nil)
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("Seen"))
		if err != nil {
			return fmt.Errorf("Error creating bucket 'Seen': %s", err)
		}
		return nil
	})
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
	handlers = append(handlers, DebugHandler)
	if roomCfg.Join {
		handlers = append(handlers, NickChangeHandler)
		handlers = append(handlers, JoinEventHandler)
		handlers = append(handlers, PartEventHandler)
	}
	inbound := make(chan *PacketEvent, 4)
	outbound := make(chan *PacketEvent, 4)
	errChan := make(chan error)
	cmdChan := make(chan string)
	return &Room{&roomData{0, make(map[string]time.Time),
		make(map[string]empty)}, roomCfg, db, handlers, time.Now(),
		inbound, outbound, errChan, sr, cmdChan}, nil
}

func (r *Room) SendPayload(payload interface{}, pType PacketType) {
	msg, err := MakePacket(strconv.Itoa(r.data.msgID), pType, payload)
	if err != nil {
		log.Printf("ERROR: making packet of type %s.", pType)
		return
	}
	go func() {
		r.outbound <- msg
	}()
	r.data.msgID++
}

// Auth sends an authentication packet with the given password.
func (r *Room) Auth(password string) {
	payload, _ := json.Marshal(AuthCommand{
		Type:     "passcode",
		Passcode: password})
	r.SendPayload(payload, AuthType)
}

// SendText sends a text message to the euphoria room.
func (r *Room) SendText(text string, parent string) {
	payload := SendCommand{
		Content: text,
		Parent:  parent}
	r.SendPayload(payload, SendType)
}

// SendPing sends a ping-reply, used in response to a ping-event.
func (r *Room) SendPing(time int64) {
	payload := PingReply{UnixTime: time}
	r.SendPayload(payload, PingReplyType)
}

// SendNick sends a nick-event, setting the bot's nickname in the room.
func (r *Room) SendNick(nick string) {
	payload := NickCommand{Name: nick}
	r.SendPayload(payload, NickType)
}

func (r *Room) storeSeen(user string, time int64) error {
	err := r.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Seen"))
		b.Put([]byte(user), []byte(strconv.FormatInt(time, 10)))
		return nil
	})
	return err
}

func (r *Room) retrieveSeen(user string) ([]byte, error) {
	var t []byte
	err := r.db.View(func(tx *bolt.Tx) error {
		t = tx.Bucket([]byte("Seen")).Get([]byte(user))
		return nil
	})
	return t, err
}

func (r *Room) dispatcher() {
	var fanout [](chan PacketEvent)
	var cmdChans [](chan string)
	for i, h := range r.handlers {
		fanout = append(fanout, make(chan PacketEvent, 4))
		cmdChans = append(cmdChans, make(chan string))
		go h(r, fanout[i], cmdChans[i])
	}
	for {
		select {
		case inboundMsg := <-r.inbound:
			for _, channel := range fanout {
				channel <- *inboundMsg
			}
		case cmd := <-r.cmdChan:
			fmt.Println("Killing workers...")
			for _, channel := range cmdChans {
				channel <- cmd
			}
			return
		case err := <-r.errChan:
			panic(err)
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
	if err := r.sr.Connect(r.sr.GetRoom()); err != nil {
		panic(err)
	}
	go r.sr.Receiver(r.inbound)
	go r.sr.Sender(r.outbound)
	r.dispatcher()
}

func (r *Room) Stop() {
	fmt.Println("Initiating Stop()")
	r.cmdChan <- "kill"
	r.sr.Stop()
	time.Sleep(time.Duration(1000) * time.Millisecond)
	fmt.Println("Finished stop.")
}

func (r *Room) UserLeaving(user string) bool {
	if _, ok := r.data.userLeaving[user]; ok {
		return true
	}
	return false
}

func (r *Room) ClearUserLeaving(user string) {
	delete(r.data.userLeaving, user)
}

func (r *Room) SetUserLeaving(user string) {
	r.data.userLeaving[user] = empty{}
}
