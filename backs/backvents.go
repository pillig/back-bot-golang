package backs

import (
	"back-bot/backs/loot"
	"back-bot/backs/model"
	"fmt"
	"io/fs"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type BackInfo struct {
	VoiceState *discordgo.VoiceState
	// TODO: is Message used for anything?
	Message *discordgo.MessageCreate
	Back    *discordgo.User
}

type BackHandler interface {
	OnBack(s *discordgo.Session, m *discordgo.MessageCreate)
}

type backHandlerLootActions interface {
	AddLoot(userID loot.UserID, loot model.Back)
	Rollback(userID loot.UserID)
}

type backHandler struct {
	backfs      fs.FS
	backs       BackMapping
	lootActions backHandlerLootActions
}

var _ MessageHandler = new(backHandler) // *backHandler implements MessageHandler

func NewBackHandler(backfs fs.FS, provider BackProvider) (*backHandler, error) {
	return &backHandler{
		backfs: backfs,
		backs:  provider.Backs(),
	}, nil
}

func (b *backHandler) ConnectLootActions(la backHandlerLootActions) {
	b.lootActions = la
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

			vs, err := retrieveVoiceStateForPlayback(s, m.Author.ID, m.ChannelID)
			if err != nil {
				return false, fmt.Errorf("BackHandler: error retrieving voice state for playback: %w", err)
			}

			if vs == nil {
				fmt.Printf("detected back, but user was not found in voice channel. username: %v\n", m.Author.Username)
				return true, nil
			}

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

	return false, nil
}

func retrieveVoiceStateForPlayback(s *discordgo.Session, originatingUserID string, channelID string) (*discordgo.VoiceState, error) {
	// Find where that Back came from.
	c, err := s.State.Channel(channelID)
	if err != nil {
		return nil, err
	}

	// Find the guild for that channel.
	g, err := s.State.Guild(c.GuildID)
	if err != nil {
		return nil, err
	}

	// Look for the message sender in that guild's current voice states.
	for _, vs := range g.VoiceStates {
		if vs.UserID == originatingUserID {
			return vs, nil
		}
	}

	return nil, nil
}
