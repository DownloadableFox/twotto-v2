package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	BotToken string `json:"bot_token"`
	Colors   struct {
		Info    int `json:"info"`
		Result  int `json:"result"`
		Success int `json:"success"`
		Warning int `json:"warning"`
		Error   int `json:"error"`
	} `json:"colors"`
}

type ConfigProvider interface {
	GetConfig() (Config, error)
}

type JsonConfigProvider struct{}

func NewJsonConfigProvider() ConfigProvider {
	return &JsonConfigProvider{}
}

func (c *JsonConfigProvider) GetConfig() (Config, error) {
	data, err := os.ReadFile("config/config.json")
	if err != nil {
		return Config{}, err
	}

	var config Config
	if err = json.Unmarshal(data, &config); err != nil {
		return Config{}, err
	}

	return config, nil
}
