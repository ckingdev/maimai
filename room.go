package maimai

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
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
	MsgLog       bool
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
	Logger   *logrus.Logger
	wg       sync.WaitGroup
}

func (r *Room) StoreMsgLogEvent(msgID string, msg *MsgLogEvent) {
	data, _ := json.Marshal(msg)
	err := r.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("MsgLog"))
		b.Put([]byte(msgID), data)
		return nil
	})
	if err != nil {
		r.Logger.Errorf("Error logging message: %s", err)
	}
}

// NewRoom creates a new room with the given configurations.
func NewRoom(roomCfg *RoomConfig, room string, sr SenderReceiver, logger *logrus.Logger) (*Room, error) {
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
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("MsgLog"))
		if err != nil {
			return fmt.Errorf("Error creating bucket 'MsgLog': %s", err)
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
	if roomCfg.MsgLog {
		handlers = append(handlers, MessageLogHandler)
	}
	inbound := make(chan *PacketEvent, 4)
	outbound := make(chan *PacketEvent, 4)
	errChan := make(chan error)
	cmdChan := make(chan string)
	return &Room{&roomData{0, make(map[string]time.Time),
		make(map[string]empty)}, roomCfg, db, handlers, time.Now(),
		inbound, outbound, errChan, sr, cmdChan, logger, sync.WaitGroup{}}, nil
}

func (r *Room) SendPayload(payload interface{}, pType PacketType) {
	msg, err := MakePacket(strconv.Itoa(r.data.msgID), pType, payload)
	if err != nil {
		r.Logger.Errorf("Error sending payload type %s: %v", pType, payload)
	}
	go func() {
		r.outbound <- msg
	}()
	r.data.msgID++
}

// Auth sends an authentication packet with the given password.
func (r *Room) Auth(password string) {
	payload := AuthCommand{
		Type:     "passcode",
		Passcode: password}
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
		cmdChans = append(cmdChans, make(chan string, 1))
		r.wg.Add(1)
		go func(hd Handler, msgCh chan PacketEvent, cmdCh chan string) {
			defer r.wg.Done()
			hd(r, msgCh, cmdCh)
		}(h, fanout[i], cmdChans[i])
	}
	for {
		select {
		case inboundMsg := <-r.inbound:
			for _, channel := range fanout {
				channel <- *inboundMsg
			}
		case cmd := <-r.cmdChan:
			for _, channel := range cmdChans {
				channel <- cmd
			}
			return
		case err := <-r.errChan:
			r.Logger.Fatalf("Unhandled error received from handler: %s\n", err)
		}
	}
}

// Run provides a method for setup and the main loop that the bot will run with handlers.
func (r *Room) Run() {
	if err := r.sr.Connect(); err != nil {
		panic(err)
	}
	go r.sr.Start(r.inbound, r.outbound)
	r.dispatcher()
}

func (r *Room) Stop() {
	r.cmdChan <- "kill"
	r.sr.Stop()
	r.wg.Wait()
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
