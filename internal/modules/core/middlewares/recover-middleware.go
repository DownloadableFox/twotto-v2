package middlewares

import (
	"bytes"
	"context"
	"fmt"
	"runtime"

	"github.com/DownloadableFox/twotto-v2/internal/api"
	"github.com/bwmarrin/discordgo"
	"github.com/rs/xid"
	"github.com/rs/zerolog"
)

var _ api.CommandMiddleware = (*RecoverMiddleware)(nil)

type RecoverMiddleware struct {
	logger zerolog.Logger
}

func NewRecoverMiddleware(parent zerolog.Logger) *RecoverMiddleware {
	return &RecoverMiddleware{
		logger: parent.With().Str("middleware", "recover").Logger(),
	}
}

func (r *RecoverMiddleware) Handle(command api.Command, next api.CommandExecuteFunc) api.CommandExecuteFunc {
	return func(c context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
		defer r.PanicWrap(s, i)

		if err := next(c, s, i); err != nil {
			r.logger.Error().Err(err).Msgf("Caught an error while executing interaction \"%s\"!", command.Data().Name)

			// Reply to the interaction with an error embed
			errorEmbed := r.CreateErrorEmbed(err, xid.New()) // Generate embed
			if err := r.AttemptReply(s, i, errorEmbed); err != nil {
				r.logger.Warn().Err(err).Msg("Failed to reply to interaction!")
			}
		}

		return nil
	}
}

func (r *RecoverMiddleware) PanicWrap(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if rec := recover(); rec != nil {
		// Generate the stacktrace
		stacktrace := make([]byte, 4096)
		count := runtime.Stack(stacktrace, false)
		reader := bytes.NewReader(stacktrace[:count])

		// Print stacktrace
		r.logger.Error().Any("panic", rec).Msg("Recovered from panic in command execution")
		r.logger.Debug().Msg(string(stacktrace[:count]))

		// Generate embed
		id := xid.New()
		errorEmbed := r.CreateFatalErrorEmbed(id)

		if err := r.AttemptReply(s, i, errorEmbed); err != nil {
			r.logger.Warn().Err(err).Msg("Failed to reply to interaction!")
		}

		s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Flags: discordgo.MessageFlagsEphemeral,
			Files: []*discordgo.File{
				{
					Name:        fmt.Sprintf("st-%s.txt", id),
					ContentType: "text/plain",
					Reader:      reader,
				},
			},
		})
	}
}

func (r *RecoverMiddleware) CreateErrorEmbed(err error, id xid.ID) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Color:       api.ColorError,
		Title:       "Oh no! :(",
		Description: "Sorry! An unexpected error occurred while executing this event.\nIf this keeps happening contact <@556132236697665547>.",
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "Error Message",
				Value: err.Error(),
			},
			{
				Name:   "Error ID",
				Value:  fmt.Sprintf("`%s`", id),
				Inline: true,
			},
			{
				Name:   "Server Time",
				Value:  fmt.Sprintf("<t:%d:f>", id.Time().Unix()),
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "The following error was reported!",
		},
	}
}

func (r *RecoverMiddleware) CreateFatalErrorEmbed(id xid.ID) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Color:       api.ColorError,
		Title:       "Fatal! -w-",
		Description: "You have encountered a fatal error! This should never happen.\nIf this keeps happening contact <@556132236697665547>.",
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Error ID",
				Value:  fmt.Sprintf("`%s`", id),
				Inline: true,
			},
			{
				Name:   "Server Time",
				Value:  fmt.Sprintf("<t:%d:f>", id.Time().Unix()),
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "The following error was reported!",
		},
	}
}

func (r *RecoverMiddleware) AttemptReply(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed) error {
	// Attempt to reply
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	}); err == nil { // If reply succesful finish.
		return nil
	}

	// Check if the interaction has already been replied to
	res, err := s.InteractionResponse(i.Interaction)
	if err != nil {
		return err
	}

	// Edit reply where possible
	if res.Flags&discordgo.MessageFlagsLoading > 0 {
		if _, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{embed},
		}); err != nil {
			return err
		}
	} else {
		// If already replied, send follow-up
		if _, err := s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Flags:  res.Flags & discordgo.MessageFlagsEphemeral,
			Embeds: []*discordgo.MessageEmbed{embed},
		}); err != nil {
			return err
		}
	}

	return nil
}
