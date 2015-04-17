package main

import (
	"runtime"
	"time"

	"github.com/apologue-dot-net/maimai"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU() - 1)
	botCfg := &maimai.BotConfig{"errors.log"}
	roomCfg := &maimai.RoomConfig{"MaiMai", ""}
	connCfg := &maimai.ConnConfig{"test", 5, time.Duration(1) * time.Second}

	conn, err := maimai.NewConn(connCfg)
	if err != nil {
		panic(err)
	}
	room, err := maimai.NewRoom(roomCfg, conn)

	b, err := maimai.NewBot(room, botCfg)
	if err != nil {
		panic(err)
	}
	b.Run()
}
