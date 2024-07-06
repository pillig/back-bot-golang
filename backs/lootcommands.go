package backs

import (
	"back-bot/backs/loot"
	"back-bot/backs/model"
	"fmt"
	"io/fs"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var backpackCmd = &discordgo.ApplicationCommand{
	Name:        "backpack",
	Description: "View your backpack",
	Type:        discordgo.ChatApplicationCommand,
}
var playbackCmd = &discordgo.ApplicationCommand{
	Name:        "playback",
	Description: "Play one of the backs from your backpack",
	Type:        discordgo.ChatApplicationCommand,
}
var rollbackCmd = &discordgo.ApplicationCommand{
	Name:        "rollback",
	Description: "It's time to go back to the way things were",
	Type:        discordgo.ChatApplicationCommand,
}

type LootCommands interface {
	RegisterCommands(s *discordgo.Session) error
	Backpack(s *discordgo.Session, i *discordgo.InteractionCreate)
	Playback(s *discordgo.Session, i *discordgo.InteractionCreate)
	Rollback(s *discordgo.Session, i *discordgo.InteractionCreate)
}

type lootCmdHandler struct {
	lootBag loot.LootBag
	backfs  fs.FS
}

func NewLootCmdHandler(lb loot.LootBag, backfs fs.FS) *lootCmdHandler {
	return &lootCmdHandler{
		lootBag: lb,
		backfs:  backfs,
	}
}

var _ LootCommands = new(lootCmdHandler) // *lootCmdHandler implements LootCommands

func (l *lootCmdHandler) Backpack(s *discordgo.Session, i *discordgo.InteractionCreate) {
	user := i.Member.User
	if user == nil {
		// in DM context, User is populated instead of Member. yeah I don't know.
		user = i.User
	}

	userID := loot.UserID(user.ID)
	userState := l.lootBag.GetState(userID)

	lootByRarity := userState.LootByRarity()

	var content strings.Builder
	w := func(s string, args ...any) { fmt.Fprintf(&content, s, args...) }
	wln := func(s string, args ...any) { s = s + "\n"; w(s, args...) }

	// TODO: improvements to message content:
	//   * empty states
	//   * color for different rarities
	//   * more compact columnar layout (look at https://pkg.go.dev/text/tabwriter)
	wln("")
	wln("%s's loot:", user.Username)
	wln("Rare:")
	for _, lootItem := range lootByRarity[model.Rare] {
		wln("🔙 %s: %d", lootItem.Back.Filename(), lootItem.Count)
	}

	wln("")
	wln("Uncommon:")
	for _, lootItem := range lootByRarity[model.Uncommon] {
		wln("🔙 %s: %d", lootItem.Back.Filename(), lootItem.Count)
	}

	wln("")
	wln("Common:")
	for _, lootItem := range lootByRarity[model.Common] {
		wln("🔙 %s: %d", lootItem.Back.Filename(), lootItem.Count)
	}

	resp := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: content.String(),
		},
	}

	err := s.InteractionRespond(i.Interaction, resp)
	if err != nil {
		// TODO: structured logging
		fmt.Printf("error responding to /backpack command: %v\n", err)
	}
}

func (l *lootCmdHandler) Playback(s *discordgo.Session, i *discordgo.InteractionCreate) {

}

func (l *lootCmdHandler) Rollback(s *discordgo.Session, i *discordgo.InteractionCreate) {

}

// RegisterCommands should be called on the bot's Session to initially register the commands
// and appropriate handlers.
func (l *lootCmdHandler) RegisterCommands(s *discordgo.Session) error {
	_, err := s.ApplicationCommandCreate(s.State.User.ID, "", backpackCmd)
	if err != nil {
		return fmt.Errorf("failed to create backpackCmd: %w", err)
	}

	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		fmt.Printf("handling an interaction! name: %s\n", i.ApplicationCommandData().Name)
		// TODO: better delegation by name
		if i.ApplicationCommandData().Name == "backpack" {
			l.Backpack(s, i)
			return
		}
	})

	return nil
}
