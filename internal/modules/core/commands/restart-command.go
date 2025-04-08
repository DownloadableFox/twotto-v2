package commands

import (
	"context"
	"errors"
	"os"
	"slices"

	"github.com/DownloadableFox/twotto-v2/internal/api"
	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog"
)

var _ api.Command = (*RestartCommand)(nil)

type RestartCommand struct {
	logger zerolog.Logger
}

func NewRestartCommand(parent zerolog.Logger) *RestartCommand {
	return &RestartCommand{
		logger: parent.With().Str("command", "restart").Logger(),
	}
}

// Data implements api.Command.
func (r *RestartCommand) Data() discordgo.ApplicationCommand {
	return discordgo.ApplicationCommand{
		Name:        "restart",
		Description: "Restarts the bot.",
	}
}

// Execute implements api.Command.
func (r *RestartCommand) Execute(c context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	var owners = []string{"556132236697665547", "836684190987583576", "610825796285890581"}

	// Get the user
	var userId string
	if i.Member != nil {
		userId = i.Member.User.ID
	} else {
		userId = i.User.ID
	}

	// Check if the user is an owner
	if !slices.Contains(owners, userId) {
		return errors.New("you are not authorized to use this command")
	}

	// Send a response
	response := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "Restarting...",
					Color:       api.ColorInfo,
					Description: "The bot is now restarting. Please wait a moment.",
				},
			},
		},
	}

	if err := s.InteractionRespond(i.Interaction, response); err != nil {
		return err
	}

	// Log the restart
	r.logger.Warn().Msg("Restart command received- Restarting bot...")

	// Close the session
	os.Exit(0)

	return nil
}
