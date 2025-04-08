package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DownloadableFox/twotto-v2/internal/api"
	"github.com/DownloadableFox/twotto-v2/internal/config"
	"github.com/DownloadableFox/twotto-v2/internal/modules/core"
	"github.com/DownloadableFox/twotto-v2/internal/modules/yiff"
	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}

func main() {
	configProvider := config.NewJsonConfigProvider()

	// Load the config using the provider
	config, err := configProvider.GetConfig()
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	// Initialize the bot with the loaded config
	client, err := discordgo.New("Bot " + config.BotToken)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create Discord client!")
	}

	// Set the bot's presence
	client.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildMessageReactions
	client.StateEnabled = true
	client.Compress = true

	// Command Manager
	moduleManager := api.NewModuleManager()
	commandManager := api.NewCommandManager()
	eventManager := api.NewEventManager()

	// Register the core module
	log.Info().Msg("Registering modules ...")
	if err := moduleManager.RegisterModules(
		core.NewCoreModule(log.Logger),
		yiff.NewYiffModule(log.Logger),
	); err != nil {
		log.Fatal().Err(err).Msg("Failed to register module!")
	}

	// Register events
	log.Info().Msg("Registering events ...")
	if err := moduleManager.OnEvents(client, eventManager); err != nil {
		log.Fatal().Err(err).Msg("Failed to register events!")
	}

	// Run the bot until terminated
	if err := client.Open(); err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Discord!")
	}
	defer client.Close()

	// Register commands
	log.Info().Msg("Registering commands ...")
	if err := moduleManager.OnCommands(client, commandManager); err != nil {
		log.Fatal().Err(err).Msg("Failed to register commands!")
	}

	log.Info().Msg("Bot is set and running!")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}
