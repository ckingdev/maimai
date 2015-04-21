package main

import (
	"runtime"
	"time"

	"github.com/apologue-dot-net/maimai"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU() - 1)
	roomCfg := &maimai.RoomConfig{"MaiMai", "", "testroom.db", "errors.log", []maimai.Handler{}}
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
	b, err := maimai.NewBot(room, botCfg)
	if err != nil {
		panic(err)
	}
	b.Room.SendNick(roomCfg.Nick)
	b.Run()
}
