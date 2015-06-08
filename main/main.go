package main

import (
	"flag"
	"os"
	"runtime"

	"github.com/cpalone/maimai"

	"github.com/Sirupsen/logrus"
)

var roomName string
var nick string
var logPath string
var dbPath string
var password string
var join bool
var msgLog bool
var logger = logrus.New()

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

	logFile, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	defer logFile.Close()
	logger.Out = logFile
	logger.Level = logrus.DebugLevel
	logger.Formatter = &logrus.JSONFormatter{}

	runtime.GOMAXPROCS(runtime.NumCPU() - 1)
	roomCfg := &maimai.RoomConfig{
		DBPath:       dbPath,
		ErrorLogPath: logPath,
		Join:         join,
		MsgLog:       msgLog,
		Nick:         nick,
		Password:     password,
	}
	room, err := maimai.NewRoom(roomCfg, roomName, maimai.NewWSSenderReceiver(roomName, logger), logger)
	if err != nil {
		panic(err)
	}
	room.Run()
}
