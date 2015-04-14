package maimai

import (
	"errors"
)

func PingEventHandler(bot *Bot, packet *PacketEvent) error {
	if packet.Type != PingType {
		return nil
	}
	payload, err := packet.Payload()
	if err != nil {
		return err
	}
	data, ok := payload.(*PingEvent)
	if !ok {
		return errors.New("Unable to assert payload as PingEvent.")
	}
	err = bot.Room.SendPing(data.Time)
	if err != nil {
		return err
	}
	return nil
}

func PingCommandHandler(bot *Bot, packet *PacketEvent) error {
	if packet.Type != SendType {
		return nil
	}
	payload, err := packet.Payload()
	if err != nil {
		return err
	}
	data, ok := payload.(*SendEvent)
	if !ok {
		return errors.New("Unable to assert payload as SendEvent.")
	}
	if len(data.Content) >= 5 && data.Content[0:5] == "!ping" {
		err = bot.Room.SendText("pong!", data.ID)
		return err
	}
	return nil
}
