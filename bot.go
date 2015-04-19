package maimai

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
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
	err = bot.Room.SendPing(data.Time)
	if err != nil {
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
		err = bot.Room.SendText("pong!", data.ID)
		if err != nil {
			log.Printf("ERROR: %s\n", err)
		}
	}
	return
}

// SeenRecordHandler handles a send-event and records that the sender was seen.
func SeenRecordHandler(bot *Bot, packet *PacketEvent) {
	if packet.Type != SendType {
		return
	}
	// TODO : refactor these two commands into a function?
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
	if data.Sender.Name != "" {
		bot.Room.data.seen[strings.Replace(data.Sender.Name, " ", "", -1)] = time.Now()
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
		lastSeen, ok := bot.Room.data.seen[splits[1][1:]]
		if !ok {
			bot.Room.SendText("User has not been seen yet.", data.ID)
			return
		}
		since := time.Since(lastSeen)
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
	//b.Room.SendNick(b.Room.config.Nick)
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
