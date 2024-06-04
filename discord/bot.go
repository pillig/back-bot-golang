package discord

import (
	"back-bot/backs"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	Session  *discordgo.Session
	BackList *backs.BackMapping
}

func NewBot(token string) *Bot {
	session, err := discordgo.New(fmt.Sprintf("Bot %s", token))
	if err != nil {
		fmt.Println("Could not authenticate Back Bot with Discord")
		return nil
	}

	backList, err := backs.GetBacks()
	if err != nil {
		fmt.Println("Problem getting the Back files back")
		return nil
	}

	return &Bot{
		Session:  session,
		BackList: backList,
	}
}

func (b Bot) Open() error {
	return b.Session.Open()
}

func (b Bot) Close() {
	b.Session.Close()
}

func (b Bot) AddHandler(handler interface{}) {
	b.Session.AddHandler(handler)
}

func (b Bot) Start() {
	b.Session.AddHandler(backs.OnBack)
	// We need information about guilds (which includes their channels),
	// messages and voice states.
	b.Session.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildVoiceStates
}
