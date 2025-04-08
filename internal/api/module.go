package api

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type Module interface {
	Events() ([]EventStack, error)
	Tasks() ([]TaskStack, error)
	Commands() ([]CommandStack, error)
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
	// Register events
	for _, module := range m.Modules {
		events, err := module.Events()
		if err != nil {
			return fmt.Errorf("failed to factory events for module %T: %w", module, err)
		}

		for _, stack := range events {
			if err := manager.RegisterStack(stack); err != nil {
				return fmt.Errorf("failed to register event for module %T: %w", module, err)
			}
		}
	}

	// Publish events
	if err := manager.PublishEvents(client); err != nil {
		return fmt.Errorf("failed to publish events: %w", err)
	}

	return nil
}

func (m *ModuleManager) OnCommands(client *discordgo.Session, manager CommandManager) error {
	// Register commands
	for _, module := range m.Modules {
		commands, err := module.Commands()
		if err != nil {
			return fmt.Errorf("failed to factory commands for module %T: %w", module, err)
		}

		for _, stack := range commands {
			if err := manager.RegisterStack(stack); err != nil {
				return fmt.Errorf("failed to register command for module %T: %w", module, err)
			}
		}
	}

	// Publish commands
	if err := manager.PublishCommands(client); err != nil {
		return fmt.Errorf("failed to publish commands: %w", err)
	}

	return nil
}

func (m *ModuleManager) OnTasks(client *discordgo.Session, manager TaskManager) error {
	// Register tasks
	for _, module := range m.Modules {
		tasks, err := module.Tasks()
		if err != nil {
			return fmt.Errorf("failed to factory tasks for module %T: %w", module, err)
		}

		for _, stack := range tasks {
			if err := manager.RegisterStack(stack); err != nil {
				return fmt.Errorf("failed to register task for module %T: %w", module, err)
			}
		}
	}

	// Publish tasks
	if err := manager.PublishTasks(client); err != nil {
		return fmt.Errorf("failed to publish tasks: %w", err)
	}

	return nil
}
