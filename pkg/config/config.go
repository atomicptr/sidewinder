package config

import (
	"io"
	"log"
	"time"

	"github.com/BurntSushi/toml"
)

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

func (c *Config) FindGroup(name string) Group {
	for _, g := range c.Groups {
		if g.Name == name {
			return g
		}
	}

	log.Fatalf("could not find group: %s", name)
	return Group{}
}
