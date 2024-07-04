package main

import (
	"back-bot/discord"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

func init() {
	flag.StringVar(&token, "t", "", "Bot Token")
	flag.StringVar(&tokenFile, "f", "", "Bot Token File")
	flag.Parse()
}

var token string
var tokenFile string

func main() {

	if token == "" {
		if tokenFile == "" {
			fmt.Println("No token or file provided. Please run: go run main.go -t <bot token> or Please run: go run main.go -f <token file>")
			return
		} else {
			file, err := os.ReadFile(tokenFile)
			if err != nil {
				fmt.Println("Could not open token file")
				return
			}
			token = string(file)
		}
	}

	bot := discord.NewBot(token)
	if bot == nil {
		fmt.Println("Back bot could not be started")
		return
	}
	bot.Start()

	// We need information about guilds (which includes their channels),
	// messages and voice states.
	bot.Session.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildVoiceStates

	// Open the websocket and begin listening.
	err := bot.Open()
	if err != nil {
		fmt.Println("Error opening Discord session: ", err)
		os.Exit(1)
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Back bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session.
	bot.Close()

}
