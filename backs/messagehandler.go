package backs

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type MessageHandler interface {
	Handle(session *discordgo.Session, msg *discordgo.MessageCreate) (handled bool, err error)
}

// MessageDelegator is a MessageHandler that delegates the message
// to its list of handlers, halting once one reports having handled it.
type MessageDelegator struct {
	Handlers []MessageHandler
}

func NewMessageDelegator(handlers ...MessageHandler) *MessageDelegator {
	return &MessageDelegator{
		Handlers: handlers,
	}
}

func (m *MessageDelegator) Handle(s *discordgo.Session, msg *discordgo.MessageCreate) (handled bool, err error) {
	for _, handler := range m.Handlers {
		handled, err = handler.Handle(s, msg)
		if err != nil {
			// TODO: structured logging
			fmt.Printf("MessageDelegator: error from delegatee. msg: %+v  err: %v \n", msg.Message, err)
		}
		if handled {
			return
		}
	}

	return false, nil
}
