package discord

import (
	"back-bot/backs"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	Session     *discordgo.Session
	BackHandler backs.BackHandler
}

// FIXME: May not want this hardcoded forever!
const backRepoPath = "back_repo"

func NewBot(token string) *Bot {
	session, err := discordgo.New(fmt.Sprintf("Bot %s", token))
	if err != nil {
		fmt.Println("Could not authenticate Back Bot with Discord")
		return nil
	}

	backHandler, err := backs.NewBackHandler(backRepoPath)
	if err != nil {
		fmt.Println("Failed to instantiate backHandler")
		return nil
	}

	return &Bot{
		Session:     session,
		BackHandler: backHandler,
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
	b.Session.AddHandler(b.BackHandler.OnBack)
	// We need information about guilds (which includes their channels),
	// messages and voice states.
	b.Session.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildVoiceStates
}
