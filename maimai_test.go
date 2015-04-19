package maimai

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestPingResponse(t *testing.T) {
	inbound := make(chan PacketEvent, 1)
	outbound := make(chan []byte, 1)
	mockConn := mockConnection{&inbound, &outbound}
	room, err := NewRoom(&RoomConfig{"MaiMai", ""}, mockConn)
	if err != nil {
		panic(err)
	}
	b, err := NewBot(room, &BotConfig{"testing.log"})
	if err != nil {
		panic(err)
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
		panic(err)
	}
	packet.Data = serial
	inbound <- packet
	time.Sleep(time.Second)
	pingReplyI := <-outbound
	var pingReplyPE PacketEvent
	if err = json.Unmarshal(pingReplyI, &pingReplyPE); err != nil {
		panic(err)
	}
	if pingReplyPE.Type != "ping-reply" {
		fmt.Println(pingReplyPE.Type)
		panic("Type of ping reply is not 'ping-reply'.")
	}
	var payloadReply PingEvent
	if err = json.Unmarshal(pingReplyPE.Data, &payloadReply); err != nil {
		panic(err)
	}
	if payloadReply.Time != timeSent {
		panic("Mismatch between time in inbound and outbound packets.")
	}
}
