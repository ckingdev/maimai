package maimai

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func NewTestBot() (*Bot, *chan PacketEvent, *chan []byte) {
	inbound := make(chan PacketEvent, 1)
	outbound := make(chan []byte, 1)
	mockConn := mockConnection{&inbound, &outbound}
	room, err := NewRoom(&RoomConfig{"MaiMai", ""}, mockConn)
	if err != nil {
		panic(fmt.Sprintf("Error creating room: %s\n", err))
	}
	b, err := NewBot(room, &BotConfig{"testing.log"})
	if err != nil {
		panic(fmt.Sprintf("Error creating bot: %s\n", err))
	}
	return b, &inbound, &outbound
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

func TestPingResponse(t *testing.T) {
	b, inbound, outbound := NewTestBot()
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
		t.Fatalf("Type of nick packet is not 'nick'. Expected nick, got %s",
			nickPacketEvent.Type)
	}
	nickPayload := make(map[string]string)
	if err = json.Unmarshal(nickPacketEvent.Data, &nickPayload); err != nil {
		t.Fatalf("Error unmarshalling nick payload: %s\n", err)
	}
	if nick, ok := nickPayload["name"]; ok {
		if nick != "MaiMai" {
			t.Fatalf("Incorrect nick. Expected MaiMai, got %s\n", nick)
		}
	} else {
		t.Fatal("'nick' not found as payload field.\n")
	}
}

func TestTextSend(t *testing.T) {
	inbound := make(chan PacketEvent, 1)
	outbound := make(chan []byte, 1)
	mockConn := mockConnection{&inbound, &outbound}
	room, err := NewRoom(&RoomConfig{"MaiMai", ""}, mockConn)
	if err != nil {
		t.Fatalf("Error creating room: %s\n", err)
	}
	room.SendText("test text", "parent")
	textPacketRaw := <-outbound
	_, textPayload := ReceiveSendPacket(textPacketRaw)
	if text, ok := (*textPayload)["content"]; ok {
		if text != "test text" {
			t.Fatalf("Incorrect text. Expected 'test text', got '%s'\n", text)
		}
	} else {
		t.Fatal("'content' not found as payload field.")
	}
}

func TestPingCommand(t *testing.T) {
	bot, inbound, outbound := NewTestBot()
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
		t.Fatalf("Error marshalling send-event: %s\n", err)
	}
	pingPacket := PacketEvent{ID: "0",
		Type: "send-event",
		Data: data}
	*inbound <- pingPacket
	pongData := <-*outbound
	_, pongPayload := ReceiveSendPacket(pongData)
	if text, ok := (*pongPayload)["content"]; ok {
		if text != "pong!" {
			t.Fatalf("Reply is not 'pong!'. Got %s\n", text)
		}
	} else {
		t.Fatal("No content field in payload.")
	}
	if parent, ok := (*pongPayload)["parent"]; ok {
		if parent != "1" {
			t.Fatalf("Incorrect parent. Expected 1, got %s\n", parent)
		}
	} else {
		t.Fatal("No parent field in payload.")
	}
}
