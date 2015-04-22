package main

import (
	"runtime"
	"time"
	"flag"

	"github.com/apologue-dot-net/maimai"
)

var roomName string
var nick string
var logPath string
var dbPath string

func init() {
	const (
		defaultRoom = "test"
		defaultNick = "MaiMai"
		defaultLog = "room_test.log"
		defaultDB = "room_test.db"
	)
	flag.StringVar(&roomName, "room", defaultRoom, "room for the bot to join")
	flag.StringVar(&nick, "nick", defaultNick, "nick for the bot to use")
	flag.StringVar(&logPath, "log", defaultLog, "path for the bot's log")
	flag.StringVar(&dbPath, "db", defaultDB, "path for the bot's db")
}

func main() {
	flag.Parse()
	runtime.GOMAXPROCS(runtime.NumCPU() - 1)
	roomCfg := &maimai.RoomConfig{"MaiMai", "", "testroom.db", "errors.log"}
	connCfg := &maimai.ConnConfig{"test", 5, time.Duration(1) * time.Second}

	conn, err := maimai.NewConn(connCfg)
	if err != nil {
		panic(err)
	}
	room, err := maimai.NewRoom(roomCfg, conn)
	if err != nil {
		panic(err)
	}
	err = room.SendNick("MaiMai")
	room.SendNick(roomCfg.Nick)
	room.Run()
}
