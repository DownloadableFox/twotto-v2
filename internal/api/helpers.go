package api

import (
	"errors"

	"github.com/bwmarrin/discordgo"
)

// Command data helper
var ErrOptionNotFound = errors.New("option not found")
var ErrOptionUnexpectedType = errors.New("unexpected option type")

func GetStringOption(data []*discordgo.ApplicationCommandInteractionDataOption, name string) (string, error) {
	for _, option := range data {
		if option.Name == name {
			if option.Type != discordgo.ApplicationCommandOptionString {
				return "", ErrOptionUnexpectedType
			}

			return option.StringValue(), nil
		}
	}

	return "", ErrOptionNotFound
}

func GetStringDefaultOption(data []*discordgo.ApplicationCommandInteractionDataOption, name string, def string) string {
	value, err := GetStringOption(data, name)
	if err != nil {
		return def
	}

	return value
}

func GetIntegerOption(data []*discordgo.ApplicationCommandInteractionDataOption, name string) (int, error) {
	for _, option := range data {
		if option.Name == name {
			if option.Type != discordgo.ApplicationCommandOptionInteger {
				return 0, ErrOptionUnexpectedType
			}

			return int(option.IntValue()), nil
		}
	}

	return 0, ErrOptionNotFound
}

func GetIntegerDefaultOption(data []*discordgo.ApplicationCommandInteractionDataOption, name string, def int) int {
	value, err := GetIntegerOption(data, name)
	if err != nil {
		return def
	}

	return value
}

func GetBooleanOption(data []*discordgo.ApplicationCommandInteractionDataOption, name string) (bool, error) {
	for _, option := range data {
		if option.Name == name {
			if option.Type != discordgo.ApplicationCommandOptionBoolean {
				return false, ErrOptionUnexpectedType
			}

			return option.BoolValue(), nil
		}
	}

	return false, ErrOptionNotFound
}

func GetBooleanDefaultOption(data []*discordgo.ApplicationCommandInteractionDataOption, name string, def bool) bool {
	value, err := GetBooleanOption(data, name)
	if err != nil {
		return def
	}

	return value
}
