package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type WebhookType = string

const (
	DiscordType WebhookType = "discord"
)

type Webhook struct {
	Type WebhookType `toml:"type"`
	Url  string      `toml:"url"`
}

type ItemData struct {
	Title       string
	Description string
	URL         string
	Published   time.Time
}

func (wh *Webhook) Fire(feed Feed, title, description, url string, ts time.Time) error {
	log.Printf("fire group %s: { title = %s; description = %s; url = %s; time = %v}\n", feed.Group, title, description, url, ts)

	return wh.fireDiscord(feed, title, description, url, ts)
}

func (wh *Webhook) FireBatch(feed Feed, items []ItemData) error {
	var desc string
	maxLen := 3800
	for i, item := range items {
		line := "- [" + item.Title + "](" + item.URL + ")\n"
		if len(desc)+len(line) > maxLen {
			remaining := len(items) - i
			desc += "...\n*" + fmt.Sprintf("%d more", remaining) + "*"
			break
		}
		desc += line
	}

	body := discordContent{
		Content: "",
		Embeds: []discordEmbed{
			{
				Author: discordAuthor{
					Name: "sidewinder",
					Url:  "https://github.com/atomicptr/sidewinder",
				},
				Title:       feed.Name + ": " + fmt.Sprintf("%d new posts", len(items)),
				Description: desc,
				Fields: []discordField{
					{
						Name:   "Feed",
						Value:  feed.Name,
						Inline: false,
					},
					{
						Name:   "Posts",
						Value:  fmt.Sprintf("%d", len(items)),
						Inline: true,
					},
				},
				Timestamp: time.Now().Format(time.RFC3339),
			},
		},
	}

	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	log.Println(string(data))

	return postDiscord(wh.Url, data)
}

type discordAuthor struct {
	Name string `json:"name"`
	Url  string `json:"url"`
}

type discordField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type discordEmbed struct {
	Author      discordAuthor  `json:"author"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Fields      []discordField `json:"fields"`
	Timestamp   string         `json:"timestamp"`
}

type discordContent struct {
	Content string         `json:"content"`
	Embeds  []discordEmbed `json:"embeds"`
}

func cut(str string, num int) string {
	if len(str) <= num {
		return str
	}
	return str[0:num] + "..."
}

func (wh *Webhook) fireDiscord(feed Feed, title, description, url string, ts time.Time) error {
	body := discordContent{
		Content: "",
		Embeds: []discordEmbed{
			{
				Author: discordAuthor{
					Name: "sidewinder",
					Url:  "https://github.com/atomicptr/sidewinder",
				},
				Title:       title,
				Description: cut(description, 100),
				Fields: []discordField{
					{
						Name:   "Feed",
						Value:  feed.Name,
						Inline: false,
					},
					{
						Name:   "Url",
						Value:  url,
						Inline: false,
					},
				},
				Timestamp: ts.Format(time.RFC3339),
			},
		},
	}

	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	log.Println(string(data))

	return postDiscord(wh.Url, data)
}

func postDiscord(url string, data []byte) error {
	res, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer func() {
		err := res.Body.Close()
		if err != nil {
			log.Printf("could not close webhook response body: %s\n", err)
		}
	}()

	d, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	log.Printf("%d - response: %s\n", res.StatusCode, string(d))
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("discord webhook returned %d: %s", res.StatusCode, strings.TrimSpace(string(d)))
	}

	return nil
}
