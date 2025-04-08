package extras

import (
	"fmt"

	"github.com/DownloadableFox/twotto-v2/internal/api"
	"github.com/DownloadableFox/twotto-v2/internal/modules/extras/commands"
	"github.com/DownloadableFox/twotto-v2/internal/modules/extras/middlewares"
	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog"
)

func Initialize(client *discordgo.Session, registrar api.CommandManager) error {
	// Generate middlewares
	recoverMiddleware := middlewares.NewRecoverMiddleware(&zerolog.Logger{})

	// Register the ping command
	if err := registrar.RegisterCommand(&commands.PingCommand{}, recoverMiddleware); err != nil {
		return fmt.Errorf("failed to register ping command: %w", err)
	}

	return nil
}
