package maimai

import (
	"log"
	"os"
)

type Handler func(bot *Bot, packet *PacketEvent) error

type Bot struct {
	Room        *Room
	handlers    []Handler
	DebugLogger log.Logger
	config      *BotConfig
}

type BotConfig struct {
	ErrorLogPath string
}

func NewBot(roomCfg *RoomConfig, connConfig *ConnConfig,
	botConfig *BotConfig) (*Bot, error) {
	room, err := NewRoom(roomCfg, connConfig)
	if err != nil {
		return nil, err
	}
	var bot Bot
	bot.handlers = append(bot.handlers, PingEventHandler)
	bot.handlers = append(bot.handlers, PingCommandHandler)
	bot.Room = room
	bot.config = botConfig
	return &bot, nil
}

func (b *Bot) Run() {
	if b.config.ErrorLogPath != "" {
		errorFile, err := os.OpenFile(b.config.ErrorLogPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			panic(err)
		}
		defer errorFile.Close()
		log.SetOutput(errorFile)
	}
	b.Room.SendNick(b.Room.config.Nick)
	for {
		packet, err := b.Room.conn.receivePacketWithRetries()
		if err != nil {
			panic(err)
		}
		for _, handler := range b.handlers {
			go handler(b, packet)
		}
	}
}
