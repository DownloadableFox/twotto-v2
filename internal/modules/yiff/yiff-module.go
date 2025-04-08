package yiff

import (
	"github.com/DownloadableFox/twotto-v2/internal/api"
	"github.com/DownloadableFox/twotto-v2/internal/modules/core/middlewares"
	"github.com/DownloadableFox/twotto-v2/internal/modules/yiff/commands"
	"github.com/DownloadableFox/twotto-v2/internal/modules/yiff/services"
	"github.com/DownloadableFox/twotto-v2/internal/modules/yiff/tasks"
	"github.com/rs/zerolog"
)

var _ api.Module = (*YiffModule)(nil)

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

func (m *YiffModule) Events() ([]api.EventStack, error) {
	return []api.EventStack{}, nil
}

func (m *YiffModule) Commands() ([]api.CommandStack, error) {
	return []api.CommandStack{
		api.CompileCommand(
			commands.NewYiffCommand(m.service, m.logger),
			middlewares.NewRecoverMiddleware(m.logger),
		),
	}, nil
}

func (m *YiffModule) Tasks() ([]api.TaskStack, error) {
	return []api.TaskStack{
		api.CompileTasks(
			tasks.NewPopularTask(m.logger, m.service),
		),
	}, nil
}
