package backs

import (
	"back-bot/backs/loot"
	"back-bot/backs/model"
	"fmt"
	"io/fs"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// need an addressable "false" for DMPermission field
var falseVar bool

var backpackCmd = &discordgo.ApplicationCommand{
	Name:         "backpack",
	Description:  "View your backpack",
	Type:         discordgo.ChatApplicationCommand,
	DMPermission: &falseVar,
}
var playbackCmd = &discordgo.ApplicationCommand{
	Name:         "playback",
	Description:  "Play one of the backs from your backpack",
	Type:         discordgo.ChatApplicationCommand,
	DMPermission: &falseVar,
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:         discordgo.ApplicationCommandOptionString,
			Name:         "chosen-back",
			Description:  "The back you desire to play.",
			Autocomplete: true,
			Required:     true,
		},
	},
}
var rollbackCmd = &discordgo.ApplicationCommand{
	Name:         "rollback",
	Description:  "It's time to go back to the way things were",
	Type:         discordgo.ChatApplicationCommand,
	DMPermission: &falseVar,
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
	backs   BackMapping
}

func NewLootCmdHandler(lb loot.LootBag, backfs fs.FS, provider BackProvider) *lootCmdHandler {
	return &lootCmdHandler{
		lootBag: lb,
		backfs:  backfs,
		backs:   provider.Backs(),
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
	wln("%s's loot:", user.Username)
	wln("Rare (%d each):", model.RarityLootValues[model.Rare])
	for _, lootItem := range lootByRarity[model.Rare] {
		if lootItem.Count > 0 {
			wln("🔙 %s: %d", lootItem.Back.Filename(), lootItem.Count)
		}
	}

	wln("")
	wln("Uncommon (%d each):", model.RarityLootValues[model.Uncommon])
	for _, lootItem := range lootByRarity[model.Uncommon] {
		if lootItem.Count > 0 {
			wln("🔙 %s: %d", lootItem.Back.Filename(), lootItem.Count)
		}
	}

	wln("")
	wln("Common (%d each):", model.RarityLootValues[model.Common])
	for _, lootItem := range lootByRarity[model.Common] {
		if lootItem.Count > 0 {
			wln("🔙 %s: %d", lootItem.Back.Filename(), lootItem.Count)
		}
	}

	wln("")
	wln("Total nominal value is %d greenbacks", userState.RarityPoints())

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
	// Command only allowed in channels, so user will be in Member field
	userID := loot.UserID(i.Member.User.ID)
	userState := l.lootBag.GetState(userID)
	userInput := i.ApplicationCommandData().Options[0].StringValue()

	// Handle generating and presenting autocomplete results
	if i.Type == discordgo.InteractionApplicationCommandAutocomplete {

		var choices []*discordgo.ApplicationCommandOptionChoice
		for back, count := range userState.Loot {
			if count < 1 {
				continue
			}

			if strings.Contains(back.Backname(), userInput) {
				choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
					Name:  back.Backname(),
					Value: back.Path(),
				})
			}
		}

		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionApplicationCommandAutocompleteResult,
			Data: &discordgo.InteractionResponseData{
				Choices: choices,
			},
		})
		if err != nil {
			// TODO: structured logging
			fmt.Printf("failed to send autocomplete response. username: %v input: %v err: %v\n", i.Member.User.Username, userInput, err)
		}
	}

	// Handle user's definitive selection of an option from autocomplete results
	if i.Type == discordgo.InteractionApplicationCommand {
		// userInput should be a valid back path from their loot
		back, err := model.GetBack(userInput)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags:   discordgo.MessageFlagsEphemeral,
					Content: fmt.Sprintf("%s is not a valid back path!", userInput),
				},
			})
			return
		}

		count, ok := userState.Loot[back]
		if count < 1 || !ok {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags:   discordgo.MessageFlagsEphemeral,
					Content: fmt.Sprintf("you don't appear to have %s in your backpack! back off!", back.Backname()),
				},
			})
			return
		}

		// TODO: maybe we should just do this at the end instead of having
		// logic to undo it. oh well
		l.lootBag.RemoveLoot(userID, back)

		var playbackFailed bool
		defer func() {
			if playbackFailed {
				l.lootBag.AddLoot(userID, back)
			}
		}()

		backData, err := loadBack(l.backfs, back.Path())
		if err != nil {
			playbackFailed = true
			// TODO: structured logging
			fmt.Printf("failed to load back data while handling /playback. path: %v err: %v\n", back.Path(), err)
			return
		}
		vs, err := retrieveVoiceStateForPlayback(s, i.Member.User.ID, i.ChannelID)
		if err != nil {
			playbackFailed = true
			// TODO: structured logging
			fmt.Printf("failed to retrieve voice state for playback. username: %v channelID: %v err: %v\n", i.Member.User.ID, i.ChannelID, err)
			return
		}

		if vs == nil {
			playbackFailed = true
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("Hey %s, get back in a voice channel if you want playback.", i.Member.User.Username),
				},
			})
			return
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "Yeah, I'm thinking you're back.",
			},
		})

		err = playBack(s, BackInfo{
			VoiceState: vs,
			Back:       i.Member.User,
			Message:    nil, // Message unused?
		}, backData)
		if err != nil {
			playbackFailed = true
			// TODO: structured logging
			fmt.Printf("error in playBack while handling /playback. back: %v username: %v err: %v\n", back.Filename(), i.Member.User.Username, err)
			return
		}
	}
}

func (l *lootCmdHandler) Rollback(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Command only allowed in channels, so user will be in Member field
	userID := loot.UserID(i.Member.User.ID)
	userState := l.lootBag.GetState(userID)

	rarityPoints := userState.RarityPoints()

	// TODO: structured logging
	fmt.Printf("%s wants to roll back with %d rarity points...\n", i.Member.User.Username, rarityPoints)

	if rarityPoints < 10_000 {
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags: discordgo.MessageFlagsEphemeral,
				Content: fmt.Sprintf(
					"You need more than 10,000 cumulative value in your backpack before you can rollback. "+
						"You only have %d.",
					rarityPoints,
				),
			},
		})
		if err != nil {
			// TODO: structured logging
			fmt.Printf("error sending interaction response: %v\n", err)
		}
		return
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Ooohh %s is rolling it back!!!!!", i.Member.User.Username),
		},
	})
	if err != nil {
		// TODO: structured logging
		fmt.Printf("error sending interaction response: %v\n", err)
	}

	rollback, err := pickFromBackList(l.backs, model.Rollback)
	if err != nil {
		// TODO: structured logging
		fmt.Printf("failed to pick rollback model while handling /rollback. err: %v\n", err)
		return
	}

	backData, err := loadBack(l.backfs, rollback.Path())
	if err != nil {
		// TODO: structured logging
		fmt.Printf("failed to load back data while handling /rollback. err: %v\n", err)
		return
	}
	vs, err := retrieveVoiceStateForPlayback(s, i.Member.User.ID, i.ChannelID)
	if err != nil {
		// TODO: structured logging
		fmt.Printf("failed to retrieve voice state for /rollback. username: %v channelID: %v err: %v\n", i.Member.User.ID, i.ChannelID, err)
		return
	}

	if vs == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Hey %s, you have to roll into a voice channel before I'll let you roll it back.", i.Member.User.Username),
			},
		})
		return
	}

	// Ooohhh
	l.lootBag.Rollback(userID)

	err = playBack(s, BackInfo{
		VoiceState: vs,
		Back:       i.Member.User,
		Message:    nil, // Message unused?
	}, backData)
	if err != nil {
		// TODO: structured logging
		fmt.Printf("error in playBack while handling /rollback. back: %v username: %v err: %v\n", rollback.Filename(), i.Member.User.Username, err)
		return
	}
}

// RegisterCommands should be called on the bot's Session to initially register the commands
// and appropriate handlers.
func (l *lootCmdHandler) RegisterCommands(s *discordgo.Session) error {
	_, err := s.ApplicationCommandCreate(s.State.User.ID, "", backpackCmd)
	if err != nil {
		return fmt.Errorf("failed to create backpackCmd: %w", err)
	}

	_, err = s.ApplicationCommandCreate(s.State.User.ID, "", playbackCmd)
	if err != nil {
		return fmt.Errorf("failed to create playbackCmd: %w", err)
	}

	_, err = s.ApplicationCommandCreate(s.State.User.ID, "", rollbackCmd)
	if err != nil {
		return fmt.Errorf("failed to create rollbackCmd: %w", err)
	}

	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		fmt.Printf("handling an interaction! name: %s\n", i.ApplicationCommandData().Name)

		switch i.ApplicationCommandData().Name {
		case "backpack":
			l.Backpack(s, i)
		case "playback":
			l.Playback(s, i)
		case "rollback":
			l.Rollback(s, i)
		}
	})

	return nil
}
