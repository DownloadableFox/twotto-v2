package api

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type Module interface {
	OnEvents(manager EventManager) error
	OnCommands(manager CommandManager) error
}

type ModuleManager struct {
	Modules []Module
}

func NewModuleManager() *ModuleManager {
	return &ModuleManager{
		Modules: make([]Module, 0),
	}
}

func (m *ModuleManager) RegisterModules(module ...Module) error {
	m.Modules = append(m.Modules, module...)
	return nil
}

func (m *ModuleManager) OnEvents(client *discordgo.Session, manager EventManager) error {
	for _, module := range m.Modules {
		if err := module.OnEvents(manager); err != nil {
			return fmt.Errorf("failed to register events for module %T: %w", module, err)
		}
	}

	if err := manager.PublishEvents(client); err != nil {
		return fmt.Errorf("failed to publish events: %w", err)
	}

	return nil
}

func (m *ModuleManager) OnCommands(client *discordgo.Session, manager CommandManager) error {
	for _, module := range m.Modules {
		if err := module.OnCommands(manager); err != nil {
			return fmt.Errorf("failed to register commands for module %T: %w", module, err)
		}
	}

	if err := manager.PublishCommands(client); err != nil {
		return fmt.Errorf("failed to publish commands: %w", err)
	}

	return nil
}
