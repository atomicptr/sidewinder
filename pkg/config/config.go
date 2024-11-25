package config

import (
	"io"
	"time"

	"github.com/BurntSushi/toml"
)

type WebhookType = string

const (
	DiscordType WebhookType = "discord"
)

type Webhook struct {
	Type WebhookType `toml:"type"`
	Url  string      `toml:"url"`
}

type Group struct {
	Name     string    `toml:"name"`
	Webhooks []Webhook `toml:"webhooks"`
}

type Feed struct {
	Name  string `toml:"name"`
	Url   string `toml:"url"`
	Group string `toml:"group"`
}

type Config struct {
	TickRate time.Duration `toml:"tick-rate"`

	Groups []Group `toml:"groups"`
	Feeds  []Feed  `toml:"feeds"`
}

func Read(r io.Reader) (*Config, error) {
	var config Config
	_, err := toml.NewDecoder(r).Decode(&config)

	if err != nil {
		return nil, err
	}

	// TODO: validate config

	return &config, nil
}
