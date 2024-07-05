package backs

import (
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type BackInfo struct {
	VoiceState *discordgo.VoiceState
	Message    *discordgo.MessageCreate
	Back       *discordgo.User
}

type BackHandler interface {
	OnBack(s *discordgo.Session, m *discordgo.MessageCreate)
}

type backHandler struct {
	backfs fs.FS
	backs  BackMapping
}

var _ MessageHandler = new(backHandler) // *backHandler implements MessageHandler

func NewBackHandler(repoPath string) (*backHandler, error) {
	backfs := os.DirFS(repoPath)
	backs, err := GetBacks(backfs)
	if err != nil {
		return nil, fmt.Errorf("failed to get back (the most essential action). err: %w", err)
	}

	return &backHandler{
		backfs: backfs,
		backs:  backs,
	}, nil
}

// Handle is added as a handler to the Discord bot's connection.
// It'll be called whenever a message comes through on a channel that
// the bot is monitoring.
func (b *backHandler) Handle(s *discordgo.Session, m *discordgo.MessageCreate) (bool, error) {
	fmt.Println("Message detected, checking for backs: ", m.Content)

	// Back Bot can't back itself
	if m.Author.ID == s.State.User.ID {
		return false, nil
	}

	// check if the message is a variation of "back"
	for _, word := range BackWords {
		if strings.Contains(strings.ToLower(m.Content), word) {
			fmt.Println("BACK DETECTED, PLAYING BACK")

			// Find where that Back came from.
			c, err := s.State.Channel(m.ChannelID)
			if err != nil {
				return true, fmt.Errorf("BackHandler: error finding channel. msg: %+v err: %w", m.Message, err)
			}

			// Find the guild for that channel.
			g, err := s.State.Guild(c.GuildID)
			if err != nil {
				// Could not find guild.
				return true, fmt.Errorf("BackHandler: error finding guild/server. msg: %+v err: %w", m.Message, err)
			}

			// Look for the message sender in that guild's current voice states.
			for _, vs := range g.VoiceStates {
				if vs.UserID == m.Author.ID {
					err = b.Who(s, BackInfo{
						VoiceState: vs,
						Message:    m,
						Back:       m.Author,
					})
					if err != nil {
						err = fmt.Errorf("BackHandler: error playing sound: %w", err)
					}

					return true, err
				}
			}
		}
	}

	return false, nil
}
