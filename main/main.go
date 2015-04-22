package main

import (
	"flag"
	"runtime"
	"time"

	"github.com/apologue-dot-net/maimai"
)

var roomName string
var nick string
var logPath string
var dbPath string
var password string

func init() {
	const (
		defaultRoom = "test"
		defaultNick = "MaiMai"
		defaultLog  = "room_test.log"
		defaultDB   = "room_test.db"
		defaultPass = ""
	)
	flag.StringVar(&roomName, "room", defaultRoom, "room for the bot to join")
	flag.StringVar(&nick, "nick", defaultNick, "nick for the bot to use")
	flag.StringVar(&logPath, "log", defaultLog, "path for the bot's log")
	flag.StringVar(&dbPath, "db", defaultDB, "path for the bot's db")
	flag.StringVar(&password, "pass", defaultPass, "password for the room")
}

func main() {
	flag.Parse()
	runtime.GOMAXPROCS(runtime.NumCPU() - 1)
	roomCfg := &maimai.RoomConfig{nick, "", dbPath, logPath}
	connCfg := &maimai.ConnConfig{roomName, 5, time.Duration(1) * time.Second}

	conn, err := maimai.NewConn(connCfg)
	if err != nil {
		panic(err)
	}
	room, err := maimai.NewRoom(roomCfg, conn)
	if err != nil {
		panic(err)
	}
	if password != "" {
		if err = room.Auth(password); err != nil {
			panic(err)
		}
	}
	err = room.SendNick("MaiMai")
	room.SendNick(roomCfg.Nick)
	room.Run()
}
