package events

import (
	"context"
	"fmt"

	"github.com/DownloadableFox/twotto-v2/internal/api"
	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var _ api.Event[discordgo.Ready] = (*OnReadyEvent)(nil)

type OnReadyEvent struct {
	logger zerolog.Logger
}

func NewOnReadyEvent(parent zerolog.Logger) *OnReadyEvent {
	return &OnReadyEvent{
		logger: parent.With().Str("event", "on-ready").Logger(),
	}
}

// Data implements api.Event.
func (o *OnReadyEvent) Data() api.EventData {
	return api.EventData{
		Name: "core-on-ready",
		Once: true,
	}
}

// Execute implements api.Event.
func (o *OnReadyEvent) Execute(c context.Context, s *discordgo.Session, e *discordgo.Ready) error {
	o.logger.Info().Msgf("Logged in as %s#%s", e.User.Username, e.User.Discriminator)

	// List connected guilds
	guilds, err := s.UserGuilds(0, "", "", false)
	if err != nil {
		o.logger.Warn().Err(err).Msg("Failed to list guilds")
	} else {
		if len(guilds) == 0 {
			log.Warn().Msg("Not connected to any guilds")
		} else {
			guildsGreeting := ""
			for i, guild := range guilds {
				if i > 0 {
					guildsGreeting += ", "
				}

				guildsGreeting += guild.Name
			}

			o.logger.Info().Msgf("Connected to guilds: %s", guildsGreeting)
		}
	}

	// Generate and print the bot's invite link
	inviteLink := fmt.Sprintf("https://discord.com/oauth2/authorize?client_id=%s&scope=bot&permissions=%d", s.State.User.ID, discordgo.PermissionAdministrator)
	o.logger.Info().Msgf("Bot invite link: %s", inviteLink)

	// Update status to do not disturb
	if err := s.UpdateStatusComplex(discordgo.UpdateStatusData{
		Status: "dnd",
		Activities: []*discordgo.Activity{{
			Name: "femboy furries",
			Type: discordgo.ActivityTypeWatching,
			URL:  "https://www.youtube.com/watch?v=lmSgyD5Jb_w",
		}},
	}); err != nil {
		return err
	}

	return nil
}
