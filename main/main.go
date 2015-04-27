package main

import (
	"flag"
	"runtime"

	"github.com/apologue-dot-net/maimai"
)

var roomName string
var nick string
var logPath string
var dbPath string
var password string
var join bool
var msgLog bool

func init() {
	const (
		defaultRoom   = "test"
		defaultNick   = "MaiMai"
		defaultLog    = "room_test.log"
		defaultDB     = "room_test.db"
		defaultPass   = ""
		defaultJoin   = false
		defaultMsgLog = false
	)
	flag.StringVar(&roomName, "room", defaultRoom, "room for the bot to join")
	flag.StringVar(&nick, "nick", defaultNick, "nick for the bot to use")
	flag.StringVar(&logPath, "log", defaultLog, "path for the bot's log")
	flag.StringVar(&dbPath, "db", defaultDB, "path for the bot's db")
	flag.StringVar(&password, "pass", defaultPass, "password for the room")
	flag.BoolVar(&join, "join", defaultJoin, "whether the bot sends join/part/nick messages")
	flag.BoolVar(&msgLog, "msglog", defaultMsgLog, "whether the bot logs messages.")
}

func main() {
	flag.Parse()
	runtime.GOMAXPROCS(runtime.NumCPU() - 1)
	roomCfg := &maimai.RoomConfig{nick, "", dbPath, logPath, join, msgLog}
	room, err := maimai.NewRoom(roomCfg, roomName, &maimai.WSSenderReceiver{Room: roomName})
	if err != nil {
		panic(err)
	}
	if password != "" {
		room.Auth(password)
	}
	room.SendNick(roomCfg.Nick)
	room.Run()
}
