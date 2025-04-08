package core

import (
	"fmt"

	"github.com/DownloadableFox/twotto-v2/internal/api"
	"github.com/DownloadableFox/twotto-v2/internal/modules/core/commands"
	"github.com/DownloadableFox/twotto-v2/internal/modules/core/events"
	"github.com/DownloadableFox/twotto-v2/internal/modules/core/middlewares"
	"github.com/rs/zerolog"
)

var _ api.Module = (*CoreModule)(nil)

type CoreModule struct {
	Logger zerolog.Logger
}

func NewCoreModule(parent zerolog.Logger) *CoreModule {
	return &CoreModule{
		Logger: parent.With().Str("module", "core").Logger(),
	}
}

func (c *CoreModule) OnCommands(manager api.CommandManager) error {
	// Generate middlewares
	recoverMiddleware := middlewares.NewRecoverMiddleware(c.Logger)

	// Register the ping command
	if err := manager.RegisterCommand(&commands.PingCommand{}, recoverMiddleware); err != nil {
		return fmt.Errorf("failed to register ping command: %w", err)
	}

	// Register the help command
	if err := manager.RegisterCommand(&commands.ErrorTestCommand{}, recoverMiddleware); err != nil {
		return fmt.Errorf("failed to register help command: %w", err)
	}

	// Register the restart command
	if err := manager.RegisterCommand(&commands.RestartCommand{}, recoverMiddleware); err != nil {
		return fmt.Errorf("failed to register restart command: %w", err)
	}

	return nil
}

func (c *CoreModule) OnEvents(manager api.EventManager) error {
	// Register on ready stack
	onReadyStack := api.CompileEvent(events.NewOnReadyEvent(c.Logger))
	if err := manager.RegisterStack(onReadyStack); err != nil {
		return fmt.Errorf("failed to register on ready event: %w", err)
	}

	return nil
}
