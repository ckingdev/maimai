package maimai

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// func TemplateHandler(room *Room, input chan PacketEvent, cmdChan chan string) {
// 	for {
// 		select {
// 		case packet := <-input:

// 		case cmd := <-cmdChan:
// 			if cmd == "kill" {
// 				return
// 			}
// 		}
// 	}
// }

var linkMatcher = regexp.MustCompile("(https?://)?[\\S]+\\.[\\S][\\S]+[\\S^\\.]")

// DEBUG toggles DEBUG-level logging messages.
const DEBUG = true

// Handler describes functions that process packets.
type Handler func(room *Room, input chan PacketEvent, cmdChan chan string)

// PingEventHandler processes a ping-event and replies with a ping-reply.
func PingEventHandler(room *Room, input chan PacketEvent, cmdChan chan string) {
	for {
		select {
		case packet := <-input:
			if packet.Type != PingEventType {
				continue
			}
			fmt.Println("Handling ping-event.")
			if DEBUG {
				log.Println("DEBUG: Replying to ping.")
			}
			payload, err := packet.Payload()
			if err != nil {
				log.Printf("ERROR: %s\n", err)
				room.errChan <- err
				return
			}
			data, ok := payload.(*PingEvent)
			if !ok {
				log.Println("ERROR: Unable to assert payload as *PingEvent.")
				room.errChan <- err
				return
			}
			room.SendPing(data.Time)
		case cmd := <-cmdChan:
			if cmd == "kill" {
				return
			}
		}
	}
}

func isValidPingCommand(payload *SendEvent) bool {
	if len(payload.Content) >= 5 && payload.Content[0:5] == "!ping" {
		return true
	}
	return false
}

// PingCommandHandler handles a send-event, checks for a !ping, and replies.
func PingCommandHandler(room *Room, input chan PacketEvent, cmdChan chan string) {
	for {
		select {
		case packet := <-input:
			if packet.Type != SendEventType {
				continue
			}
			payload, err := packet.Payload()
			if err != nil {
				log.Printf("ERROR: %s\n", err)
				room.errChan <- err
				return
			}
			data, ok := payload.(*SendEvent)
			if !ok {
				log.Println("ERROR: Unable to assert payload as *SendEvent.")
				room.errChan <- err
				return
			}
			if isValidPingCommand(data) {
				if DEBUG {
					log.Println("DEBUG: Handling !ping command.")
				}
				room.SendText("pong!", data.ID)
			}
		case cmd := <-cmdChan:
			if cmd == "kill" {
				return
			}
		}
	}
}

// SeenRecordHandler handles a send-event and records that the sender was seen.
func SeenRecordHandler(room *Room, input chan PacketEvent, cmdChan chan string) {
	for {
		select {
		case packet := <-input:
			if packet.Type != SendEventType {
				continue
			}
			payload, err := packet.Payload()
			if err != nil {
				log.Printf("ERROR: %s\n", err)
				room.errChan <- err
				return
			}
			data, ok := payload.(*SendEvent)
			if !ok {
				log.Println("ERROR: Unable to assert payload as *SendEvent.")
				room.errChan <- err
				return
			}
			user := strings.Replace(data.Sender.Name, " ", "", -1)
			t := time.Now().Unix()
			err = room.storeSeen(user, t)
			if err != nil {
				room.errChan <- err
				return
			}
		case cmd := <-cmdChan:
			if cmd == "kill" {
				return
			}
		}
	}
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
func SeenCommandHandler(room *Room, input chan PacketEvent, cmdChan chan string) {
	for {
		select {
		case packet := <-input:
			if packet.Type != SendEventType {
				continue
			}
			payload, err := packet.Payload()
			if err != nil {
				log.Printf("ERROR: %s\n", err)
				room.errChan <- err
				return
			}
			data, ok := payload.(*SendEvent)
			if !ok {
				log.Println("ERROR: Unable to assert payload as *SendEvent.")
				room.errChan <- err
				return
			}
			if isValidSeenCommand(data) {
				trimmed := strings.TrimSpace(data.Content)
				splits := strings.Split(trimmed, " ")
				lastSeen, err := room.retrieveSeen(splits[1][1:])
				if err != nil {
					room.errChan <- err
					return
				}
				if lastSeen == nil {
					room.SendText("User has not been seen yet.", data.ID)
					continue
				}
				lastSeenInt, err := strconv.Atoi(string(lastSeen))
				if err != nil {
					room.errChan <- err
					return
				}
				lastSeenTime := time.Unix(int64(lastSeenInt), 0)
				since := time.Since(lastSeenTime)
				room.SendText(fmt.Sprintf("Seen %v hours and %v minutes ago.\n",
					int(since.Hours()), int(since.Minutes())), data.ID)
			}
		case cmd := <-cmdChan:
			if cmd == "kill" {
				return
			}
		}
	}
}

func getLinkTitle(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Bad response code: %v", resp.StatusCode)
	}
	z := html.NewTokenizer(resp.Body)
	depth := 0
	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			return "", fmt.Errorf("No title found at url.")
		case html.TextToken:
			if depth > 0 {
				title := strings.TrimSpace(string(z.Text()))
				if title == "Imgur" {
					return "", nil
				}
				return title, nil
			}
		case html.StartTagToken:
			tn, _ := z.TagName()
			if string(tn) == "title" {
				depth++
			}
		}
	}

}

// LinkTitleHandler handles a send-event, looks for URLs, and replies with the
// title text of a link if a valid one is found.
func LinkTitleHandler(room *Room, input chan PacketEvent, cmdChan chan string) {
	for {
		select {
		case packet := <-input:
			if packet.Type != SendEventType {
				continue
			}
			payload, err := packet.Payload()
			if err != nil {
				log.Printf("ERROR: %s\n", err)
				room.errChan <- err
				return
			}
			data, ok := payload.(*SendEvent)
			if !ok {
				log.Println("ERROR: Unable to assert payload as *SendEvent.")
				room.errChan <- err
				return
			}
			urls := linkMatcher.FindAllString(data.Content, -1)
			for _, url := range urls {
				if !strings.HasPrefix(url, "http") {
					url = "http://" + url
				}
				title, err := getLinkTitle(url)
				if err == nil && title != "" {
					room.SendText("Link title: "+title, data.ID)
					break
				}
			}
		case cmd := <-cmdChan:
			if cmd == "kill" {
				return
			}
		}
	}
}

// UptimeCommandHandler handlers a send-event and if the command is given
// replies with the time since the bot was started.
func UptimeCommandHandler(room *Room, input chan PacketEvent, cmdChan chan string) {
	for {
		select {
		case packet := <-input:
			if packet.Type != SendEventType {
				continue
			}
			payload, err := packet.Payload()
			if err != nil {
				log.Printf("ERROR: %s\n", err)
				room.errChan <- err
				return
			}
			data, ok := payload.(*SendEvent)
			if !ok {
				log.Println("ERROR: Unable to assert payload as *SendEvent.")
				room.errChan <- err
				return
			}
			if data.Content == "!uptime" {
				since := time.Since(room.uptime)
				room.SendText(fmt.Sprintf(
					"This bot has been up for %s.",
					since.String()),
					data.ID)
			}
		case cmd := <-cmdChan:
			if cmd == "kill" {
				return
			}
		}
	}
}

func ScritchCommandHandler(room *Room, input chan PacketEvent, cmdChan chan string) {
	for {
		select {
		case packet := <-input:
			if packet.Type != SendEventType {
				continue
			}
			payload, err := packet.Payload()
			if err != nil {
				log.Printf("ERROR: %s\n", err)
				room.errChan <- err
				return
			}
			data, ok := payload.(*SendEvent)
			if !ok {
				log.Println("ERROR: Unable to assert payload as *SendEvent.")
				room.errChan <- err
				return
			}
			if data.Content == "!scritch" {
				room.SendText("/me bruxes",
					data.ID)
			}
		case cmd := <-cmdChan:
			if cmd == "kill" {
				return
			}
		}
	}
}

func DebugHandler(room *Room, input chan PacketEvent, cmdChan chan string) {
	for {
		select {
		case packet := <-input:
			if packet.Error != "" {
				log.Fatalf("Received %s packet containing error: %s", packet.Type, packet.Error)
			}
		case cmd := <-cmdChan:
			if cmd == "kill" {
				return
			}
		}
	}
}

func NickChangeHandler(room *Room, input chan PacketEvent, cmdChan chan string) {
	for {
		select {
		case packet := <-input:
			if packet.Type != NickEventType {
				continue
			}
			payload, err := packet.Payload()
			if err != nil {
				log.Printf("ERROR: %s\n", err)
				room.errChan <- err
				return
			}
			data, ok := payload.(*NickEvent)
			if !ok {
				log.Println("ERROR: Unable to assert payload as *NickEvent.")
				room.errChan <- err
				return
			}
			// Don't want to process joins or leaves here
			if data.From == "" || data.To == "" {
				continue
			}
			room.SendText(fmt.Sprintf("%s is now known as %s.", data.From, data.To), "")
		case cmd := <-cmdChan:
			if cmd == "kill" {
				return
			}
		}
	}
}

func PartTimer(room *Room, user string) {
	time.Sleep(time.Duration(120) * time.Second)
	if room.UserLeaving(user) {
		room.SendText(fmt.Sprintf("< %s left the room. >", user), "")
		room.ClearUserLeaving(user)
	}
}

func PartEventHandler(room *Room, input chan PacketEvent, cmdChan chan string) {
	for {
		select {
		case packet := <-input:
			if packet.Type != PartEventType {
				continue
			}
			payload, err := packet.Payload()
			if err != nil {
				log.Printf("ERROR: %s\n", err)
				room.errChan <- err
				return
			}
			data, ok := payload.(*PresenceEvent)
			if !ok {
				log.Println("ERROR: Unable to assert payload as *PresenceEvent.")
				room.errChan <- err
				return
			}
			user := data.User.Name
			room.SetUserLeaving(user)
			go PartTimer(room, user)
		case cmd := <-cmdChan:
			if cmd == "kill" {
				return
			}
		}
	}
}

func JoinEventHandler(room *Room, input chan PacketEvent, cmdChan chan string) {
	for {
		select {
		case packet := <-input:
			switch packet.Type {

			case JoinEventType:
				payload, err := packet.Payload()
				if err != nil {
					log.Printf("ERROR: %s\n", err)
					room.errChan <- err
					return
				}
				data, ok := payload.(*PresenceEvent)
				if !ok {
					log.Println("ERROR: Unable to assert payload as *PresenceEvent.")
					room.errChan <- err
					return
				}
				user := data.User.Name
				if room.UserLeaving(user) {
					room.ClearUserLeaving(user)
				} else if data.Name != "" {
					room.SendText(fmt.Sprintf("< %s joined the room. >", user), "")
				}
			case NickEventType:
				payload, err := packet.Payload()
				if err != nil {
					log.Printf("ERROR: %s\n", err)
					room.errChan <- err
					return
				}
				data, ok := payload.(*NickEvent)
				if !ok {
					log.Println("ERROR: Unable to assert payload as *PresenceEvent.")
					room.errChan <- err
					return
				}
				if data.From != "" {
					continue
				}
				user := data.To
				if user == "" {
					continue
				}
				if room.UserLeaving(user) {
					room.ClearUserLeaving(user)
				} else {
					room.SendText(fmt.Sprintf("< %s joined the room. >", user), "")
				}
			}
		case cmd := <-cmdChan:
			if cmd == "kill" {
				return
			}
		}
	}
}
