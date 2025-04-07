package extras

import (
	"fmt"

	"github.com/DownloadableFox/twotto-v2/internal/api"
	"github.com/DownloadableFox/twotto-v2/internal/modules/extras/commands"
	"github.com/bwmarrin/discordgo"
)

func Initialize(client *discordgo.Session, registrar api.CommandManager) error {
	// Register the ping command
	if err := registrar.RegisterCommand(&commands.PingCommand{}); err != nil {
		return fmt.Errorf("failed to register ping command: %w", err)
	}

	return nil
}
