package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/DownloadableFox/twotto-v2/internal/api"
	"github.com/bwmarrin/discordgo"
)

var _ api.Command = (*PingCommand)(nil)

type PingCommand struct{}

func (p *PingCommand) Data() discordgo.ApplicationCommand {
	return discordgo.ApplicationCommand{
		Name:        "ping",
		Description: "Ping the bot to check if it's alive",
	}
}

func (p *PingCommand) Execute(c context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	response := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "Pong! :3",
					Color:       api.ColorInfo,
					Description: fmt.Sprintf("I am alive and well! Server time is <t:%d:f>.", time.Now().Unix()),
				},
			},
		},
	}

	if err := s.InteractionRespond(i.Interaction, response); err != nil {
		return err
	}

	return nil
}
