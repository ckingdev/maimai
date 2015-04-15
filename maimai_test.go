package maimai

import (
	"testing"
	"time"
)

func TestIsValidSeenCommand(t *testing.T) {
	var validCmd SendEvent
	validCmd.Content = "!seen @foobar"
	if !isValidSeenCommand(&validCmd) {
		t.Error("'!seen @foobar' evaluated as an invalid command.")
	}
	var invalidCmd SendEvent
	invalidCmd.Content = "!seen foobar"
	if isValidSeenCommand(&invalidCmd) {
		t.Error("!seen foobar evaluated as a valid command.")
	}
}

func TestIsValidPingCommand(t *testing.T) {
	var validCmd SendEvent
	validCmd.Content = "!ping"
	if !isValidPingCommand(&validCmd) {
		t.Error("!ping evaluated as an invalid command.")
	}
	var invalidCmd SendEvent
	invalidCmd.Content = "!seen @foobar"
	if isValidPingCommand(&invalidCmd) {
		t.Error("!seen @foobar evaluated as a valid command.")
	}
}

func TestNewConn(t *testing.T) {
	_, err := NewConn(&ConnConfig{"test", 5, time.Duration(1) * time.Second})
	if err != nil {
		t.Fatal(err)
	}
}

func TestNewRoom(t *testing.T) {
	_, err := NewRoom(&RoomConfig{"MaiMai", "[Testing]"}, &ConnConfig{"test", 5, time.Duration(1) * time.Second})
	if err != nil {
		t.Fatal(err)
	}
}

func TestNewBot(t *testing.T) {
	_, err := NewBot(&RoomConfig{"MaiMai", "[Testing] "}, &ConnConfig{"test", 5, time.Duration(1) * time.Second}, &BotConfig{"test.log"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSendNickText(t *testing.T) {
	room, err := NewRoom(&RoomConfig{"MaiMai", "[Testing] "}, &ConnConfig{"test", 5, time.Duration(1) * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	err = room.SendNick("MaiMai | Automated Test")
	if err != nil {
		t.Fatal(err)
	}
	err = room.SendText("Automated test message.", "")
	if err != nil {
		t.Fatal(err)
	}
}

func TestBotRun(t *testing.T) {
	botCfg := &BotConfig{"errors.log"}
	roomCfg := &RoomConfig{"MaiMai", ""}
	connCfg := &ConnConfig{"test", 5, time.Duration(1) * time.Second}

	b, err := NewBot(roomCfg, connCfg, botCfg)
	if err != nil {
		t.Fatal(err)
	}
	go b.Run()
	secondBot, err := NewBot(&RoomConfig{"MaiMai2", ""}, connCfg, botCfg)
	time.Sleep(time.Second * time.Duration(5))
	secondBot.Room.SendNick("MaiMai2")
	secondBot.Room.SendText("!ping", "")
	secondBot.Room.SendText("!seen @MaiMai2", "")
	time.Sleep(time.Second * time.Duration(5))
}
