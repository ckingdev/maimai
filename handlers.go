package maimai

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

// DEBUG toggles DEBUG-level logging messages.
const DEBUG = true

// Handler describes functions that process packets.
type Handler func(room *Room, packet *PacketEvent, errChan chan error)

// PingEventHandler processes a ping-event and replies with a ping-reply.
func PingEventHandler(room *Room, packet *PacketEvent, errChan chan error) {
	if packet.Type != PingType {
		return
	}
	if DEBUG {
		log.Println("DEBUG: Replying to ping.")
	}
	payload, err := packet.Payload()
	if err != nil {
		log.Printf("ERROR: %s\n", err)
		errChan <- err
		return
	}
	data, ok := payload.(*PingEvent)
	if !ok {
		log.Println("ERROR: Unable to assert payload as *PingEvent.")
		errChan <- err
		return
	}

	if err = room.SendPing(data.Time); err != nil {
		errChan <- err
		log.Printf("ERROR: %s\n", err)
	}
	return
}

func isValidPingCommand(payload *SendEvent) bool {
	if len(payload.Content) >= 5 && payload.Content[0:5] == "!ping" {
		return true
	}
	return false
}

// PingCommandHandler handles a send-event, checks for a !ping, and replies.
func PingCommandHandler(room *Room, packet *PacketEvent, errChan chan error) {
	if packet.Type != SendType {
		return
	}
	payload, err := packet.Payload()
	if err != nil {
		log.Printf("ERROR: %s\n", err)
		errChan <- err
		return
	}
	data, ok := payload.(*SendEvent)
	if !ok {
		log.Println("ERROR: Unable to assert payload as *SendEvent.")
		errChan <- err
		return
	}
	if isValidPingCommand(data) {
		if DEBUG {
			log.Println("DEBUG: Handling !ping command.")
		}
		if err = room.SendText("pong!", data.ID); err != nil {
			log.Printf("ERROR: %s\n", err)
			errChan <- err
		}
	}
	return
}

// SeenRecordHandler handles a send-event and records that the sender was seen.
func SeenRecordHandler(room *Room, packet *PacketEvent, errChan chan error) {
	if packet.Type != SendType {
		return
	}
	payload, err := packet.Payload()
	if err != nil {
		log.Printf("ERROR: %s\n", err)
		errChan <- err
		return
	}
	data, ok := payload.(*SendEvent)
	if !ok {
		log.Println("ERROR: Unable to assert payload as *SendEvent.")
		errChan <- err
		return
	}
	user := strings.Replace(data.Sender.Name, " ", "", -1)
	t := time.Now().Unix()
	err = room.storeSeen(user, t)
	if err != nil {
		errChan <- err
	}
	return
}

func isValidSeenCommand(payload *SendEvent) bool {
	if len(payload.Content) >= 5 &&
		payload.Content[0:5] == "!seen" &&
		string(payload.Content[6]) == "@" &&
		len(strings.Split(payload.Content, " ")) == 2 {
		return true
	}
	return false
}

// SeenCommandHandler handles a send-event, checks if !seen command was given, and responds.
// TODO : make seen record a time when a user joins a room or changes their nick
func SeenCommandHandler(room *Room, packet *PacketEvent, errChan chan error) {
	if packet.Type != SendType {
		return
	}
	payload, err := packet.Payload()
	if err != nil {
		log.Printf("ERROR: %s\n", err)
		errChan <- err
		return
	}
	data, ok := payload.(*SendEvent)
	if !ok {
		log.Println("ERROR: Unable to assert payload as *SendEvent.")
		errChan <- err
		return
	}
	if isValidSeenCommand(data) {
		trimmed := strings.TrimSpace(data.Content)
		splits := strings.Split(trimmed, " ")
		lastSeen, err := room.retrieveSeen(splits[1][1:])
		if err != nil {
			errChan <- err
		}
		if lastSeen == nil {
			room.SendText("User has not been seen yet.", data.ID)
			return
		}
		lastSeenInt, err := strconv.Atoi(string(lastSeen))
		if err != nil {
			errChan <- err
			return
		}
		lastSeenTime := time.Unix(int64(lastSeenInt), 0)
		since := time.Since(lastSeenTime)
		err := room.SendText(fmt.Sprintf("Seen %v hours and %v minutes ago.\n",
			int(since.Hours()), int(since.Minutes())), data.ID)
		if err != nil {
			errChan <- err
		}
		return
	}
}
