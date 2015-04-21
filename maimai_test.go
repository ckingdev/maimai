package maimai

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func NewTestRoom() (*Room, *chan PacketEvent, *chan []byte) {
	inbound := make(chan PacketEvent, 1)
	outbound := make(chan []byte, 1)
	mockConn := mockConnection{&inbound, &outbound}
	room, err := NewRoom(&RoomConfig{"MaiMai", "", "test.db", "testing.log"}, mockConn)
	if err != nil {
		panic(fmt.Sprintf("Error creating room: %s\n", err))
	}
	return room, &inbound, &outbound
}

func ReceiveSendPacket(data []byte) (*PacketEvent, *map[string]string) {
	var textPacketEvent PacketEvent
	if err := json.Unmarshal(data, &textPacketEvent); err != nil {
		panic(fmt.Sprintf("Error unmarshalling send packet: %s\n", err))
	}
	if textPacketEvent.Type != "send" {
		panic(fmt.Sprintf("Type of send packet is not 'send'. Expected send, got %s\n",
			textPacketEvent.Type))
	}
	textPayload := make(map[string]string)
	if err := json.Unmarshal(textPacketEvent.Data, &textPayload); err != nil {
		panic(fmt.Sprintf("Error unmarshalling send payload: %s\n", err))
	}
	return &textPacketEvent, &textPayload
}

func CreateTestSendEvent(text string, parent string, ID string) PacketEvent {
	sender := User{ID: "0",
		Name:      "testUser",
		ServerID:  "0",
		ServerEra: "0"}
	payload := SendEvent{
		ID:      ID,
		Parent:  parent,
		Time:    time.Now().Unix(),
		Sender:  sender,
		Content: text}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		panic(fmt.Sprintf("Error marshalling payload: %s\n", err))
	}
	packet := PacketEvent{
		ID:   "0",
		Type: "send-event",
		Data: payloadJSON}
	return packet
}

func TestPingResponse(t *testing.T) {
	r, inbound, outbound := NewTestRoom()
	defer r.db.Close()
	go r.Run()
	timeSent := time.Now().Unix()
	packet := PacketEvent{ID: "0",
		Type: "ping-event"}
	payload := PingEvent{
		Time: timeSent,
		Next: timeSent + 30}
	serial, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Error marshalling ping-event payload: %s\n", err)
	}
	packet.Data = serial
	*inbound <- packet
	time.Sleep(time.Second)
	pingReplyI := <-*outbound
	var pingReplyPE PacketEvent
	if err = json.Unmarshal(pingReplyI, &pingReplyPE); err != nil {
		t.Fatalf("Error unmarshalling ping-reply packet-event: %s\n", err)
	}
	if pingReplyPE.Type != "ping-reply" {
		fmt.Println(pingReplyPE.Type)
		t.Fatal("Type of ping reply is not 'ping-reply'.")
	}
	var payloadReply PingEvent
	if err = json.Unmarshal(pingReplyPE.Data, &payloadReply); err != nil {
		t.Fatalf("Error unmarshalling reply payload: %s\n", err)
	}
	if payloadReply.Time != timeSent {
		t.Fatal("Mismatch between time in inbound and outbound packets.")
	}
	killPacket, err := json.Marshal(PacketEvent{Type: "kill"})
	if err != nil {
		panic(err)
	}
	*outbound <- killPacket
	time.Sleep(time.Duration(500) * time.Millisecond)
}

func TestNickSend(t *testing.T) {
	inbound := make(chan PacketEvent, 1)
	outbound := make(chan []byte, 1)
	mockConn := mockConnection{&inbound, &outbound}
	room, err := NewRoom(&RoomConfig{"MaiMai", "", "test.db", "testing.log"}, mockConn)
	defer room.db.Close()
	if err != nil {
		t.Fatalf("Error creating room: %s\n", err)
	}
	room.SendNick(room.config.Nick)
	nickPacketRaw := <-outbound
	var nickPacketEvent PacketEvent
	if err = json.Unmarshal(nickPacketRaw, &nickPacketEvent); err != nil {
		t.Fatalf("Error unmarshalling nick packet: %s\n", err)
	}
	if nickPacketEvent.Type != "nick" {
		t.Fatalf("Type of nick packet is not 'nick'. Expected nick, got %s",
			nickPacketEvent.Type)
	}
	nickPayload := make(map[string]string)
	if err = json.Unmarshal(nickPacketEvent.Data, &nickPayload); err != nil {
		t.Fatalf("Error unmarshalling nick payload: %s\n", err)
	}
	nick, ok := nickPayload["name"]
	if ok {
		if nick != "MaiMai" {
			t.Fatalf("Incorrect nick. Expected MaiMai, got %s\n", nick)
		}
	} else {
		t.Fatal("'nick' not found as paylo	ad field.\n")
	}
	killPacket, err := json.Marshal(PacketEvent{Type: "kill"})
	if err != nil {
		panic(err)
	}
	outbound <- killPacket
	time.Sleep(time.Duration(500) * time.Millisecond)
}

func TestTextSend(t *testing.T) {
	inbound := make(chan PacketEvent, 1)
	outbound := make(chan []byte, 1)
	mockConn := mockConnection{&inbound, &outbound}
	room, err := NewRoom(&RoomConfig{"MaiMai", "", "test.db", "testing.log"}, mockConn)
	defer room.db.Close()
	if err != nil {
		t.Fatalf("Error creating room: %s\n", err)
	}
	//	room.SendNick("MaiMai")
	room.SendText("test text", "parent")
	textPacketRaw := <-outbound
	_, textPayload := ReceiveSendPacket(textPacketRaw)
	text, ok := (*textPayload)["content"]
	if ok {
		if text != "test text" {
			t.Fatalf("Incorrect text. Expected 'test text', got '%s'\n", text)
		}
	} else {
		t.Fatal("'content' not found as payload field.")
	}
	killPacket, err := json.Marshal(PacketEvent{Type: "kill"})
	if err != nil {
		panic(err)
	}
	outbound <- killPacket
	time.Sleep(time.Duration(500) * time.Millisecond)
}

func TestPingCommand(t *testing.T) {
	r, inbound, outbound := NewTestRoom()
	defer r.db.Close()
	go r.Run()
	time.Sleep(time.Second)
	pingPacket := CreateTestSendEvent("!ping", "", "1")
	*inbound <- pingPacket
	pongData := <-*outbound
	_, pongPayload := ReceiveSendPacket(pongData)
	text, ok := (*pongPayload)["content"]
	if ok {
		if text != "pong!" {
			t.Fatalf("Reply is not 'pong!'. Got %s\n", text)
		}
	} else {
		t.Fatal("No content field in payload.")
	}
	parent, ok := (*pongPayload)["parent"]
	if ok {
		if parent != "1" {
			t.Fatalf("Incorrect parent. Expected 1, got %s\n", parent)
		}
	} else {
		t.Fatal("No parent field in payload.")
	}
	killPacket, err := json.Marshal(PacketEvent{Type: "kill"})
	if err != nil {
		panic(err)
	}
	*outbound <- killPacket
	time.Sleep(time.Duration(500) * time.Millisecond)
}

func TestSeenCommand(t *testing.T) {
	r, inbound, outbound := NewTestRoom()
	defer r.db.Close()
	go r.Run()
	time.Sleep(time.Second)
	seenPacket := CreateTestSendEvent("!seen @xyz", "", "1")
	*inbound <- seenPacket
	seenResp := <-*outbound
	_, seenPayload := ReceiveSendPacket(seenResp)
	text, ok := (*seenPayload)["content"]
	if ok {
		if text != "User has not been seen yet." {
			t.Fatalf("Incorrect response to '!seen xyz: expected User has not been seen yet.', got %s\n", text)
		}
	} else {
		t.Fatal("No content field in payload.")
	}
	seenPacket = CreateTestSendEvent("!seen @testUser", "", "2")
	*inbound <- seenPacket
	seenResp = <-*outbound
	_, seenPayload = ReceiveSendPacket(seenResp)
	text, ok = (*seenPayload)["content"]
	if ok {
		if text == "User has not been seen yet." {
			t.Fatal("Bot did not record that user was seen.")
		}
	} else {
		t.Fatal("No content field in payload.")
	}
	killPacket, err := json.Marshal(PacketEvent{Type: "kill"})
	if err != nil {
		panic(err)
	}
	*outbound <- killPacket
	time.Sleep(time.Duration(500) * time.Millisecond)
}
