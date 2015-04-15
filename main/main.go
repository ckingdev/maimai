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

	b, err := maimai.NewBot(roomCfg, connCfg, botCfg)
	if err != nil {
		panic(err)
	}
	b.Run()
}
