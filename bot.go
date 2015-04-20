package maimai

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/boltdb/bolt"
)

// DEBUG toggles DEBUG-level logging messages.
const DEBUG = true

// Handler describes functions that process packets.
type Handler func(bot *Bot, packet *PacketEvent)

// Bot holds a Room, logger, config, and handlers. This is the main object.
type Bot struct {
	Room     *Room
	handlers []Handler
	config   *BotConfig
}

// BotConfig holds the configuration for a Bot object.
type BotConfig struct {
	ErrorLogPath string
}

// NewBot creates a new bot with the given configurations.
func NewBot(room *Room, botConfig *BotConfig) (*Bot, error) {
	var bot Bot
	// TODO : change this to read handler config from file
	bot.handlers = append(bot.handlers, PingEventHandler)
	bot.handlers = append(bot.handlers, PingCommandHandler)
	bot.handlers = append(bot.handlers, SeenCommandHandler)
	bot.handlers = append(bot.handlers, SeenRecordHandler)
	bot.Room = room
	bot.config = botConfig
	return &bot, nil
}

// PingEventHandler processes a ping-event and replies with a ping-reply.
func PingEventHandler(bot *Bot, packet *PacketEvent) {
	if packet.Type != PingType {
		return
	}
	if DEBUG {
		log.Println("DEBUG: Replying to ping.")
	}
	payload, err := packet.Payload()
	if err != nil {
		log.Printf("ERROR: %s\n", err)
		return
	}
	data, ok := payload.(*PingEvent)
	if !ok {
		log.Println("ERROR: Unable to assert payload as *PingEvent.")
		return
	}

	if err = bot.Room.SendPing(data.Time); err != nil {
		log.Printf("ERROR: %s\n", err)
	}
	return
}

func isValidPingCommand(payload *SendEvent) bool {
	if len(payload.Content) >= 5 && payload.Content[0:5] == "!ping" {
		return true
	}
	return false
}

// PingCommandHandler handles a send-event, checks for a !ping, and replies.
func PingCommandHandler(bot *Bot, packet *PacketEvent) {
	if packet.Type != SendType {
		return
	}
	payload, err := packet.Payload()
	if err != nil {
		log.Printf("ERROR: %s\n", err)
		return
	}
	data, ok := payload.(*SendEvent)
	if !ok {
		log.Println("ERROR: Unable to assert payload as *SendEvent.")
		return
	}
	if isValidPingCommand(data) {
		if DEBUG {
			log.Println("DEBUG: Handling !ping command.")
		}

		if err = bot.Room.SendText("pong!", data.ID); err != nil {
			log.Printf("ERROR: %s\n", err)
		}
	}
	return
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

// SeenRecordHandler handles a send-event and records that the sender was seen.
func SeenRecordHandler(bot *Bot, packet *PacketEvent) {
	if packet.Type != SendType {
		return
	}
	payload, err := packet.Payload()
	if err != nil {
		log.Printf("ERROR: %s\n", err)
		return
	}
	data, ok := payload.(*SendEvent)
	if !ok {
		log.Println("ERROR: Unable to assert payload as *SendEvent.")
		return
	}
	user := strings.Replace(data.Sender.Name, " ", "", -1)
	t := time.Now().Unix()
	err = bot.Room.storeSeen(user, t)
	//	if data.Sender.Name != "" {
	//		bot.Room.data.seen[strings.Replace(data.Sender.Name, " ", "", -1)] = time.Now()
	//	}

	return
}

func isValidSeenCommand(payload *SendEvent) bool {
	if len(payload.Content) >= 5 &&
		payload.Content[0:5] == "!seen" &&
		string(payload.Content[6]) == "@" &&
		len(strings.Split(payload.Content, " ")) == 2 {
		return true
	}
	return false
}

// SeenCommandHandler handles a send-event, checks if !seen command was given, and responds.
// TODO : make seen record a time when a user joins a room or changes their nick
func SeenCommandHandler(bot *Bot, packet *PacketEvent) {
	if packet.Type != SendType {
		return
	}
	payload, err := packet.Payload()
	if err != nil {
		log.Printf("ERROR: %s\n", err)
		return
	}
	data, ok := payload.(*SendEvent)
	if !ok {
		log.Println("ERROR: Unable to assert payload as *SendEvent.")
		return
	}
	if isValidSeenCommand(data) {
		trimmed := strings.TrimSpace(data.Content)
		splits := strings.Split(trimmed, " ")
		lastSeen, err := bot.Room.retrieveSeen(splits[1][1:])
		if err != nil {
			panic(err)
		}
		if lastSeen == nil {
			bot.Room.SendText("User has not been seen yet.", data.ID)
			return
		}
		lastSeenInt, err := strconv.Atoi(string(lastSeen))
		if err != nil {
			panic(err)
		}
		lastSeenTime := time.Unix(int64(lastSeenInt), 0)
		since := time.Since(lastSeenTime)
		bot.Room.SendText(fmt.Sprintf("Seen %v hours and %v minutes ago.\n",
			int(since.Hours()), int(since.Minutes())), data.ID)
		return
	}
}

// Run provides a method for setup and the main loop that the bot will run with handlers.
func (b *Bot) Run() {
	if b.config.ErrorLogPath != "" {
		errorFile, err := os.OpenFile(b.config.ErrorLogPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			panic(err)
		}
		defer errorFile.Close()
		log.SetOutput(errorFile)
	}
	if DEBUG {
		log.Println("DEBUG: Setting nick.")
	}
	for {
		if DEBUG {
			log.Println("DEBUG: Handling packet.")
		}
		packet, err := b.Room.conn.receivePacketWithRetries()
		if err != nil {
			panic(err)
		}
		for _, handler := range b.handlers {
			go handler(b, packet)
		}
	}
}
