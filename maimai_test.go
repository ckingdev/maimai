package maimai

import (
	"testing"
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
