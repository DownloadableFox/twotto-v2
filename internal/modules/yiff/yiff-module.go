package yiff

import (
	"fmt"

	"github.com/DownloadableFox/twotto-v2/internal/api"
	"github.com/DownloadableFox/twotto-v2/internal/modules/core/middlewares"
	"github.com/DownloadableFox/twotto-v2/internal/modules/yiff/commands"
	"github.com/DownloadableFox/twotto-v2/internal/modules/yiff/services"
	"github.com/rs/zerolog"
)

type YiffModule struct {
	logger  zerolog.Logger
	service services.IE621Service
}

func NewYiffModule(parent zerolog.Logger) *YiffModule {
	logger := parent.With().Str("module", "yiff").Logger()

	return &YiffModule{
		logger:  logger,
		service: services.NewE621Service("twotto-v2", logger),
	}
}

func (c *YiffModule) OnCommands(manager api.CommandManager) error {
	// Generate middlewares
	recoverMiddleware := middlewares.NewRecoverMiddleware(c.logger)

	// Register the yiff command
	if err := manager.RegisterCommand(commands.NewYiffCommand(c.service, c.logger), recoverMiddleware); err != nil {
		return fmt.Errorf("failed to register yiff command: %w", err)
	}

	return nil
}

func (c *YiffModule) OnEvents(manager api.EventManager) error {
	return nil
}
