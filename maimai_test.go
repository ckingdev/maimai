package maimai

type TestHarness struct {
	outbound chan *interface{}
	inbound  chan *PacketEvent
}

func (th *TestHarness) AssertReceivedSendText(text string) {
	msg := <-th.outbound
	packet, ok := msg.(*PacketEvent)
	if !ok {
		panic("Could not assert message as *PacketEvent.")
	}
	if packet.Type != SendType {
		panic("Packet is not of type 'send'.")
	}
	payload, err := packet.Payload()
	if err != nil {
		panic("Could not extract packet payload.")
	}
	data, ok := payload.(SendEvent)
	if !ok {
		panic("Could not assert payload as SendEvent.")
	}
	if data.Content != text {
		panic("Message content does not match text. Expected '%s', got '%s'", text, data.Content)
	}
}
