package commands

import (
	"context"
	"errors"

	"github.com/DownloadableFox/twotto-v2/internal/api"
	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog"
)

var _ api.Command = (*ErrorTestCommand)(nil)

var ErrorTestCommandPermissions int64 = discordgo.PermissionAdministrator

type ErrorTestCommand struct {
	logger zerolog.Logger
}

func NewErrorTestCommand(parent zerolog.Logger) *ErrorTestCommand {
	return &ErrorTestCommand{
		logger: parent.With().Str("command", "error-test").Logger(),
	}
}

func (e *ErrorTestCommand) Data() discordgo.ApplicationCommand {
	return discordgo.ApplicationCommand{
		Name:                     "error-test",
		Description:              "Development command for testing error handling",
		DefaultMemberPermissions: &ErrorTestCommandPermissions,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "no-reply",
				Description: "Throws error before sending a reply.",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
			},
			{
				Name:        "reply",
				Description: "Throws error after replying to interaction.",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Name:        "ephemeral",
						Description: "Whether or not the reply should be ephemeral.",
						Type:        discordgo.ApplicationCommandOptionBoolean,
					},
				},
			},
			{
				Name:        "defered",
				Description: "Defer a response before throwing an error.",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Name:        "ephemeral",
						Description: "Whether or not the defer should be ephemeral.",
						Type:        discordgo.ApplicationCommandOptionBoolean,
					},
				},
			},
			{
				Name:        "panic",
				Description: "This will generate a panic in the bot, this option will not reply an error.",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
			},
		},
	}
}

func (e *ErrorTestCommand) Execute(c context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	data := i.ApplicationCommandData()

	options := data.Options
	switch options[0].Name {
	case "reply":
		var flags discordgo.MessageFlags
		if len(options[0].Options) == 0 || options[0].Options[0].BoolValue() {
			flags |= discordgo.MessageFlagsEphemeral
		}

		response := &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags: flags,
				Embeds: []*discordgo.MessageEmbed{
					{
						Title:       "Meow! :3",
						Color:       api.ColorResult,
						Description: "This is a funny & quirky response! Totally not going to die in the next 2 nanoseconds. An error is about to occur after this, depending on the handling something might or not happen.",
					},
				},
			},
		}

		if err := s.InteractionRespond(i.Interaction, response); err != nil {
			return err
		}
	case "defered":
		var flags discordgo.MessageFlags
		if len(options[0].Options) == 0 || options[0].Options[0].BoolValue() {
			flags |= discordgo.MessageFlagsEphemeral
		}

		response := &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags: flags,
			},
		}

		if err := s.InteractionRespond(i.Interaction, response); err != nil {
			return err
		}
	case "panic":
		response := &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags: discordgo.MessageFlagsEphemeral,
				Embeds: []*discordgo.MessageEmbed{
					{
						Title:       "Welp this hurts!",
						Color:       api.ColorResult,
						Description: "A panic is going to happen in my runtime in the next instants. Please beware that if unhandled correctly this might make me despawn (exit on failure) which wouldn't be optimal.",
					},
				},
			},
		}

		if err := s.InteractionRespond(i.Interaction, response); err != nil {
			return err
		}

		panic("This is a fake panic! Comming from error test command.")
	}

	return errors.New("this is a made up error")
}
