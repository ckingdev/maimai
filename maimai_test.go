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
	_, err := NewBot(&RoomConfig{"MaiMai", "[Testing]"}, &ConnConfig{"test", 5, time.Duration(1) * time.Second}, &BotConfig{"test.log"})
	if err != nil {
		t.Fatal(err)
	}
}
