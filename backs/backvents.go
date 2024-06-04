package backs

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type BackInfo struct {
	VoiceState *discordgo.VoiceState
	Message    *discordgo.MessageCreate
	Back       *discordgo.User
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func OnBack(s *discordgo.Session, m *discordgo.MessageCreate) {
	fmt.Println("Message detected, checking for backs: ", m.Content)
	// Back Bot can't back itself
	if m.Author.ID == s.State.User.ID {
		return
	}
	// check if the message is a variation of "back"
	for _, word := range BackWords {
		if strings.Contains(strings.ToLower(m.Content), word) {
			fmt.Println("BACK DETECTED, PLAYING BACK")

			// Find where that Back came from.
			c, err := s.State.Channel(m.ChannelID)
			if err != nil {
				return
			}

			// Find the guild for that channel.
			g, err := s.State.Guild(c.GuildID)
			if err != nil {
				// Could not find guild.
				return
			}

			// Look for the message sender in that guild's current voice states.
			for _, vs := range g.VoiceStates {
				if vs.UserID == m.Author.ID {
					backList, err := GetBacks()
					if err != nil {
						fmt.Println("Error getting backs:", err)
					}
					err = Who(s, BackInfo{
						VoiceState: vs,
						Message:    m,
						Back:       m.Author,
					}, backList)
					if err != nil {
						fmt.Println("Error playing sound:", err)
					}

					return
				}
			}
		}
	}

}
