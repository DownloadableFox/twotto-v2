package api

import (
	"context"
	"errors"
	"runtime"

	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"
)

type ExecuteFunction func(c context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error
type MiddlewareFunction func(command Command, next ExecuteFunction) ExecuteFunction

type CommandStack struct {
	Command    Command
	Middleware []CommandMiddleware
}

type Command interface {
	Data() discordgo.ApplicationCommand
	Execute(c context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error
}

type CommandMiddleware interface {
	Handle(command Command, next ExecuteFunction) ExecuteFunction
}

type CommandManager interface {
	PublishCommands(session *discordgo.Session) error
	RegisterCommand(cmd Command, middleware ...CommandMiddleware) error
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
		execute ExecuteFunction
	}

	var publishChannel = make(chan GeneratedCommand, len(cm.commands))

	go func() {
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

		close(publishChannel)
	}()

	// TODO: Multi-thread this to allow for multiple commands to be published at once
	// Wait for all commands to be published
	registeredCommands := make(map{string]*discordgo.Command)
	for cmd := range publishChannel {
		data := cmd.data
		next := cmd.execute

		// Register the command with the Discord API
		if command, err := session.ApplicationCommandCreate(session.State.User.ID, "", &data); err != nil {
			return err
		} else {
			registeredCommands = append(registeredCommands, command)
		}

		// Register the command handler with the session
		session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

func (cm *CommandManagerImpl) RegisterCommand(cmd Command, middleware ...CommandMiddleware) error {
	data := cmd.Data()

	if _, exists := cm.commands[data.Name]; exists {
		return errors.New("command already registered")
	}

	cm.commands[data.Name] = CommandStack{
		Command:    cmd,
		Middleware: middleware,
	}

	return nil
}
