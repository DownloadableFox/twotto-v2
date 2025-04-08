package api

import (
	"context"
	"errors"

	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron"
	"github.com/rs/zerolog/log"
)

type TaskExecuteFunc func(ctx context.Context, s *discordgo.Session) error
type TaskMiddlewareFunc func(task Task, next TaskExecuteFunc) TaskExecuteFunc

type TaskStack struct {
	Task       Task
	Middleware []TaskMiddleware
}

type TaskData struct {
	Name string
	Cron string
}

type Task interface {
	Data() TaskData
	Run(ctx context.Context, s *discordgo.Session) error
}

type TaskMiddleware interface {
	Handle(task Task, next TaskExecuteFunc) TaskExecuteFunc
}

type TaskManager interface {
	PublishTasks(session *discordgo.Session) error
	RegisterStack(stack TaskStack) error
}

type TaskManagerImpl struct {
	cron  *cron.Cron
	tasks map[string]TaskStack
}

func NewTaskManager() *TaskManagerImpl {
	return &TaskManagerImpl{
		cron:  cron.New(),
		tasks: make(map[string]TaskStack),
	}
}

func (tm *TaskManagerImpl) RegisterStack(stack TaskStack) error {
	data := stack.Task.Data()

	if _, exists := tm.tasks[data.Name]; exists {
		return errors.New("task already registered")
	}

	tm.tasks[data.Name] = stack
	return nil
}

func (tm *TaskManagerImpl) PublishTasks(session *discordgo.Session) error {
	type GeneratedTask struct {
		data    TaskData
		execute TaskExecuteFunc
	}

	var publishChannel = make(chan GeneratedTask, len(tm.tasks))

	go func() {
		defer close(publishChannel)

		for _, stack := range tm.tasks {
			data := stack.Task.Data()

			// Register the task with the middleware
			next := func(c context.Context, s *discordgo.Session) error {
				return stack.Task.Run(c, s)
			}

			// Execute the middleware in reverse order
			// to ensure the first middleware is executed last
			for i := len(stack.Middleware) - 1; i >= 0; i-- {
				mw := stack.Middleware[i]
				next = mw.Handle(stack.Task, next)
			}

			publishChannel <- GeneratedTask{
				data:    data,
				execute: next,
			}
		}
	}()

	for task := range publishChannel {
		tm.cron.AddFunc(task.data.Cron, func() {
			log.Debug().Msgf("Scheduling task %q with cron expression %q", task.data.Name, task.data.Cron)

			if err := task.execute(context.Background(), session); err != nil {
				log.Error().Err(err).Msgf("Error executing task %q not handled!", task.data.Name)
			}
		})
	}

	// Start the cron scheduler
	tm.cron.Start()
	log.Info().Msg("Cron scheduler started!")

	return nil
}

func CompileTasks(task Task, middleware ...TaskMiddleware) TaskStack {
	return TaskStack{
		Task:       task,
		Middleware: middleware,
	}
}
