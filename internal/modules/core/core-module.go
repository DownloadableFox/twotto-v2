package core

import (
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

func (m *CoreModule) Events() ([]api.EventStack, error) {
	return []api.EventStack{
		api.CompileEvent(
			events.NewOnReadyEvent(m.Logger),
		),
	}, nil
}

func (m *CoreModule) Commands() ([]api.CommandStack, error) {
	middlewares := []api.CommandMiddleware{
		middlewares.NewRecoverMiddleware(m.Logger),
	}

	return []api.CommandStack{
		api.CompileCommand(
			commands.NewPingCommand(m.Logger),
			middlewares...,
		),
		api.CompileCommand(
			commands.NewRestartCommand(m.Logger),
			middlewares...,
		),
		api.CompileCommand(
			commands.NewErrorTestCommand(m.Logger),
			middlewares...,
		),
	}, nil
}
