package discord

import (
	"back-bot/backs"
	"back-bot/backs/loot"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	Session        *discordgo.Session
	MessageHandler backs.MessageHandler
}

// FIXME: May not want this hardcoded forever!
const backRepoPath = "back_repo"

type NewBotInput struct {
	Token            string
	CsvLootStoreFile string
}

func NewBot(input NewBotInput) *Bot {
	session, err := discordgo.New(fmt.Sprintf("Bot %s", input.Token))
	if err != nil {
		fmt.Println("Could not authenticate Back Bot with Discord")
		return nil
	}

	var lootBag loot.LootBag
	if input.CsvLootStoreFile != "" {
		lootBag, err = loot.NewCsvLootBag(input.CsvLootStoreFile)
		if err != nil {
			fmt.Printf("failed to create csv loot bag. err: %v\n", err)
			return nil
		}
	}

	backHandler, err := backs.NewBackHandler(backRepoPath)
	if err != nil {
		fmt.Println("Failed to instantiate backHandler")
		return nil
	}

	backHandler.ConnectLootActions(lootBag)

	return &Bot{
		Session:        session,
		MessageHandler: backs.NewMessageDelegator(backHandler),
	}
}

func (b Bot) Open() error {
	return b.Session.Open()
}

func (b Bot) Close() {
	b.Session.Close()
}

// RootHandler calls b.MessageHandler.Handle and logs any of its errors
func (b Bot) RootHandler(s *discordgo.Session, msg *discordgo.MessageCreate) {
	_, err := b.MessageHandler.Handle(s, msg)
	if err != nil {
		fmt.Printf("Bot.RootHandler received error from MessageHandler. msg: %+v err: %v\n", msg.Message, err)
	}
}

func (b Bot) Start() {
	b.Session.AddHandler(b.RootHandler)
	// We need information about guilds (which includes their channels),
	// messages and voice states.
	b.Session.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildVoiceStates
}
