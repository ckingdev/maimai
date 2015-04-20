package maimai

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

// TODO : Change all calls of panic to t.Fatal(f)

func TestPingResponse(t *testing.T) {
	inbound := make(chan PacketEvent, 1)
	outbound := make(chan []byte, 1)
	mockConn := mockConnection{&inbound, &outbound}
	room, err := NewRoom(&RoomConfig{"MaiMai", ""}, mockConn)
	if err != nil {
		t.Fatalf("Error creating room: %s\n", err)
	}
	b, err := NewBot(room, &BotConfig{"testing.log"})
	if err != nil {
		t.Fatalf("Error creating bot: %s\n", err)
	}
	go b.Run()
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
	inbound <- packet
	time.Sleep(time.Second)
	pingReplyI := <-outbound
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
}

func TestNickSend(t *testing.T) {
	inbound := make(chan PacketEvent, 1)
	outbound := make(chan []byte, 1)
	mockConn := mockConnection{&inbound, &outbound}
	room, err := NewRoom(&RoomConfig{"MaiMai", ""}, mockConn)
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
		t.Fatal("Type of nick packet is not 'nick'. Expected nick, got %s",
			nickPacketEvent.Type)
	}
	nickPayload := make(map[string]string)
	if err = json.Unmarshal(nickPacketEvent.Data, &nickPayload); err != nil {
		t.Fatalf("Error unmarshalling nick payload: %s\n", err)
	}
	if nick, ok := nickPayload["name"]; ok {
		if nick != "MaiMai" {
			t.Fatal("Incorrect nick. Expected MaiMai, got %s", nick)
		}
	} else {
		t.Fatal("'nick' not found as payload field.")
	}
}

func TestTextSend(t *testing.T) {
	inbound := make(chan PacketEvent, 1)
	outbound := make(chan []byte, 1)
	mockConn := mockConnection{&inbound, &outbound}
	room, err := NewRoom(&RoomConfig{"MaiMai", ""}, mockConn)
	if err != nil {
		panic(err)
	}
	room.SendText("test text", "parent")
	textPacketRaw := <-outbound
	var textPacketEvent PacketEvent
	if err = json.Unmarshal(textPacketRaw, &textPacketEvent); err != nil {
		panic(err)
	}
	if textPacketEvent.Type != "send" {
		panic("Type of send packet is not 'send'.")
	}
	textPayload := make(map[string]string)
	if err = json.Unmarshal(textPacketEvent.Data, &textPayload); err != nil {
		panic(err)
	}
	if text, ok := textPayload["content"]; ok {
		if text != "test text" {
			panic("Incorrect text.")
		}
	} else {
		panic("'content' not found as payload field.")
	}
}

func TestPingCommand(t *testing.T) {
	inbound := make(chan PacketEvent, 1)
	outbound := make(chan []byte, 1)
	mockConn := mockConnection{&inbound, &outbound}
	room, err := NewRoom(&RoomConfig{"MaiMai", ""}, mockConn)
	if err != nil {
		panic(err)
	}
	bot, err := NewBot(room, &BotConfig{"testing.log"})
	if err != nil {
		panic(err)
	}
	go bot.Run()
	time.Sleep(time.Second)
	user := User{"0", "test", "test", "test"}
	pingPayload := Message{ID: "1",
		Parent:  "",
		Time:    0,
		Sender:  user,
		Content: "!ping"}
	data, err := json.Marshal(pingPayload)
	if err != nil {
		panic(err)
	}
	pingPacket := PacketEvent{ID: "0",
		Type: "send-event",
		Data: data}
	inbound <- pingPacket
	pongData := <-outbound
	var pongPacket PacketEvent
	if err = json.Unmarshal(pongData, &pongPacket); err != nil {
		panic(err)
	}
	if pongPacket.Type != "send" {
		panic("Type of send packet is not 'send'.")
	}
	pongPayload := make(map[string]string)
	if err = json.Unmarshal(pongPacket.Data, &pongPayload); err != nil {
		panic(err)
	}
	if text, ok := pongPayload["content"]; ok {
		if text != "pong!" {
			panic("Reply is not 'pong!'.")
		}
	} else {
		panic("No content field in payload.")
	}
	if parent, ok := pongPayload["parent"]; ok {
		if parent != "1" {
			panic("Incorrect parent.")
		}
	} else {
		panic("No parent field in payload.")
	}
}
