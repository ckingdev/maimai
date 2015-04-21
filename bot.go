package maimai

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DEBUG toggles DEBUG-level logging messages.
const DEBUG = true

// Handler describes functions that process packets.
type Handler func(room *Room, packet *PacketEvent, errChan chan error)

// Bot holds a Room, logger, config, and handlers. This is the main object.
type Bot struct {
	Room *Room
}

// NewBot creates a new bot with the given configurations.
func NewBot(room *Room) (*Bot, error) {
	bot := Bot{room}
	// TODO : change this to read handler config from file
	bot.Room.handlers = append(bot.Room.handlers, PingEventHandler)
	bot.Room.handlers = append(bot.Room.handlers, PingCommandHandler)
	bot.Room.handlers = append(bot.Room.handlers, SeenCommandHandler)
	bot.Room.handlers = append(bot.Room.handlers, SeenRecordHandler)
	return &bot, nil
}

// PingEventHandler processes a ping-event and replies with a ping-reply.
func PingEventHandler(room *Room, packet *PacketEvent, errChan chan error) {
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

	if err = room.SendPing(data.Time); err != nil {
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
func PingCommandHandler(room *Room, packet *PacketEvent, errChan chan error) {
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

		if err = room.SendText("pong!", data.ID); err != nil {
			log.Printf("ERROR: %s\n", err)
		}
	}
	return
}

// SeenRecordHandler handles a send-event and records that the sender was seen.
func SeenRecordHandler(room *Room, packet *PacketEvent, errChan chan error) {
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
	err = room.storeSeen(user, t)
	if err != nil {
		errChan <- err
	}
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
func SeenCommandHandler(room *Room, packet *PacketEvent, errChan chan error) {
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
		lastSeen, err := room.retrieveSeen(splits[1][1:])
		if err != nil {
			errChan <- err
		}
		if lastSeen == nil {
			room.SendText("User has not been seen yet.", data.ID)
			return
		}
		lastSeenInt, err := strconv.Atoi(string(lastSeen))
		if err != nil {
			panic(err)
		}
		lastSeenTime := time.Unix(int64(lastSeenInt), 0)
		since := time.Since(lastSeenTime)
		room.SendText(fmt.Sprintf("Seen %v hours and %v minutes ago.\n",
			int(since.Hours()), int(since.Minutes())), data.ID)
		return
	}
}

// Run provides a method for setup and the main loop that the bot will run with handlers.
func (b *Bot) Run() {
	if b.Room.config.ErrorLogPath != "" {
		errorFile, err := os.OpenFile(b.Room.config.ErrorLogPath,
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
		packet, err := b.Room.conn.receivePacketWithRetries()
		if err != nil {
			panic(err)
		}
		if packet.Type == "kill" {
			return
		}
		var wg sync.WaitGroup
		for _, handler := range b.Room.handlers {
			wg.Add(1)
			go func(h Handler) {
				defer wg.Done()
				h(b.Room, packet, errChan)
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
