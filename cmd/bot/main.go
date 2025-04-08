package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/DownloadableFox/twotto-v2/internal/api"
	"github.com/DownloadableFox/twotto-v2/internal/config"
	"github.com/DownloadableFox/twotto-v2/internal/modules/extras"
	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"
)

func main() {
	configProvider := config.NewJsonConfigProvider()

	// Load the config using the provider
	config, err := configProvider.GetConfig()
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	// Command Manager
	commandManager := api.NewCommandManager()

	// Initialize the bot with the loaded config
	client, err := discordgo.New("Bot " + config.BotToken)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create Discord client!")
	}

	// Run the bot until terminated
	if err := client.Open(); err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Discord!")
	}
	defer client.Close()

	// Set the bot's presence
	client.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildMessageReactions
	client.StateEnabled = true
	client.Compress = true

	log.Info().Msg("Initializing modules ...")

	// Register the command manager with the Discord session
	if err := extras.Initialize(client, commandManager); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize extras module!")
	}

	// Register the command manager with the Discord session
	log.Info().Msg("Publishing commands ...")

	// Register the command manager with the Discord session
	if err := commandManager.PublishCommands(client); err != nil {
		log.Fatal().Err(err).Msg("Failed to publish commands!")
	}

	// Generate and print the bot's invite link
	botID := client.State.User.ID
	permissions := discordgo.PermissionManageMessages | discordgo.PermissionSendMessages
	inviteLink := fmt.Sprintf("https://discord.com/oauth2/authorize?client_id=%s&scope=bot&permissions=%d", botID, permissions)
	log.Info().Msgf("Bot invite link: %s", inviteLink)

	log.Info().Msg("Bot is set and running!")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}
