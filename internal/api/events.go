package api

import (
	"context"
	"errors"
	"runtime"

	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"
)

type EventExecuteFunc[T any] func(c context.Context, s *discordgo.Session, e *T) error
type EventMiddlewareFunc[T any] func(event Event[T], next EventExecuteFunc[T]) EventExecuteFunc[T]

type EventStack struct {
	Data    EventData
	Execute any
}

type EventData struct {
	Name string
	Once bool
}

type Event[T any] interface {
	Data() EventData
	Execute(c context.Context, s *discordgo.Session, e *T) error
}

type EventMiddleware[T any] interface {
	Handle(event Event[T], next EventExecuteFunc[T]) EventExecuteFunc[T]
}

type EventManager interface {
	PublishEvents(session *discordgo.Session) error
	RegisterStack(event EventStack) error
}

type EventManagerImpl struct {
	events map[string]EventStack
}

func NewEventManager() *EventManagerImpl {
	return &EventManagerImpl{
		events: make(map[string]EventStack),
	}
}

func (em *EventManagerImpl) PublishEvents(session *discordgo.Session) error {
	// Wait for all events to be published
	for _, stack := range em.events {
		data := stack.Data
		execute := stack.Execute

		// Register the event handler with the session
		if data.Once {
			session.AddHandlerOnce(execute)
		} else {
			session.AddHandler(execute)
		}
	}

	return nil
}

func (em *EventManagerImpl) RegisterStack(stack EventStack) error {
	data := stack.Data

	// Register the event
	if _, exists := em.events[data.Name]; exists {
		return errors.New("event already registered")
	} else {
		em.events[data.Name] = stack
	}

	return nil
}

func CompileEvent[T any](event Event[T], middlewares ...EventMiddleware[T]) EventStack {
	data := event.Data()

	next := func(c context.Context, s *discordgo.Session, e *T) error {
		return event.Execute(c, s, e)
	}

	// Execute the middleware in reverse order
	// to ensure the first middleware is executed last
	for i := len(middlewares) - 1; i >= 0; i-- {
		mw := middlewares[i]
		next = mw.Handle(event, next)
	}

	return EventStack{
		Data:    data,
		Execute: WrapEvent(next),
	}
}

// Helper function that translates generic events into interfaces for discordgo
func WrapEvent[T any](fn EventExecuteFunc[T]) interface{} {
	return func(s *discordgo.Session, e *T) {
		defer func() {
			if rec := recover(); rec != nil {
				stacktrace := make([]byte, 4096)
				count := runtime.Stack(stacktrace, false)

				log.Error().Any("panic", rec).Msg("Recovered from fatal error while executing event!")
				log.Debug().Msg("Stack trace: \n" + string(stacktrace[:count]))
			}
		}()

		if err := fn(context.Background(), s, e); err != nil {
			log.Error().Err(err).Msg("Error executing event not handled!")
		}
	}
}
