package api

import (
	"context"
	"errors"
	"fmt"
	"runtime"

	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"
)

type CommandExecuteFunc func(c context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error
type CommandMiddlewareFunc func(command Command, next CommandExecuteFunc) CommandExecuteFunc

type CommandStack struct {
	Command    Command
	Middleware []CommandMiddleware
}

type Command interface {
	Data() discordgo.ApplicationCommand
	Execute(c context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error
}

type CommandMiddleware interface {
	Handle(command Command, next CommandExecuteFunc) CommandExecuteFunc
}

type CommandManager interface {
	PublishCommands(session *discordgo.Session) error
	RegisterStack(stack CommandStack) error
}

type CommandManagerImpl struct {
	commands map[string]CommandStack
}

func NewCommandManager() *CommandManagerImpl {
	return &CommandManagerImpl{
		commands: make(map[string]CommandStack),
	}
}

func (cm *CommandManagerImpl) PublishCommands(session *discordgo.Session) error {
	type GeneratedCommand struct {
		data    discordgo.ApplicationCommand
		execute CommandExecuteFunc
	}

	var publishChannel = make(chan GeneratedCommand, len(cm.commands))

	// Flush commands before publishing new ones
	if err := cm.FlushCommands(session); err != nil {
		return fmt.Errorf("failed to flush commands: %w", err)
	}

	go func() {
		defer close(publishChannel)

		for _, stack := range cm.commands {
			data := stack.Command.Data()

			// Register the command with the middleware
			next := func(c context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
				return stack.Command.Execute(c, s, i)
			}

			// Execute the middleware in reverse order
			// to ensure the first middleware is executed last
			for i := len(stack.Middleware) - 1; i >= 0; i-- {
				mw := stack.Middleware[i]
				next = mw.Handle(stack.Command, next)
			}

			// Send the command to the publish channel
			publishChannel <- GeneratedCommand{
				data:    data,
				execute: next,
			}
		}
	}()

	// TODO: Multi-thread this to allow for multiple commands to be published at once
	// Wait for all commands to be published
	for cmd := range publishChannel {
		data := cmd.data
		next := cmd.execute

		// Register the command with the Discord API
		if _, err := session.ApplicationCommandCreate(session.State.User.ID, "", &data); err != nil {
			return err
		}

		// Register the command handler with the session
		session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.ApplicationCommandData().Name != data.Name {
				return
			}

			defer func() {
				if rec := recover(); rec != nil {
					// Get stacktrace
					stacktrace := make([]byte, 4096)
					count := runtime.Stack(stacktrace, false)

					log.Error().Any("panic", rec).Msg("Recovered from panic in command execution")
					log.Debug().Msg("Panic stack trace: \n" + string(stacktrace[:count]))
				}
			}()

			if err := next(context.Background(), s, i); err != nil {
				log.Error().Err(err).Msg("Unhandled error in command execution")
			}
		})
	}

	return nil
}

func (cm *CommandManagerImpl) FlushCommands(session *discordgo.Session) error {
	var flushChannel = make(chan string, len(cm.commands))

	go func() {
		defer close(flushChannel)

		// Get currently registered commands
		registeredCommands, err := session.ApplicationCommands(session.State.User.ID, "")
		if err != nil {
			log.Warn().Err(err).Msg("Failed to fetch registered commands")
			return
		}

		// Check if the command is registered
		for _, cmd := range registeredCommands {
			if _, exists := cm.commands[cmd.Name]; !exists {
				flushChannel <- cmd.ID
			}
		}
	}()

	// Unregister the commands
	for cmdID := range flushChannel {
		if err := session.ApplicationCommandDelete(session.State.User.ID, "", cmdID); err != nil {
			return fmt.Errorf("failed to delete command %q: %w", cmdID, err)
		}
	}

	return nil
}

func (cm *CommandManagerImpl) RegisterStack(stack CommandStack) error {
	data := stack.Command.Data()

	if _, exists := cm.commands[data.Name]; exists {
		return errors.New("command already registered")
	}

	cm.commands[data.Name] = stack
	return nil
}

func CompileCommand(command Command, middleware ...CommandMiddleware) CommandStack {
	return CommandStack{
		Command:    command,
		Middleware: middleware,
	}
}
