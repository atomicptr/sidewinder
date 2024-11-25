package service

import (
	"fmt"
	"log"
	"time"

	"github.com/atomicptr/sidewinder/pkg/config"

	"github.com/mmcdole/gofeed"
)

func Run(config *config.Config, dataDir string) error {
	fmt.Println(config)

	ticker := time.NewTicker(config.TickRate)

	err := tick(config, dataDir)
	if err != nil {
		return err
	}

	for range ticker.C {
		err := tick(config, dataDir)
		if err != nil {
			return err
		}
	}

	return nil
}

func tick(config *config.Config, dataDir string) error {
	fp := gofeed.NewParser()

	for _, feed := range config.Feeds {
		log.Printf("fetching %s: %s...\n", feed.Name, feed.Url)

		f, err := fp.ParseURL(feed.Url)
		if err != nil {
			log.Printf("feed error %s: %s: %s", feed.Name, feed.Url, err)
			continue
		}

		newItems := filterNewItems(dataDir, f)

		if len(newItems) == 0 {
			log.Printf("feed %s: %s has no new items", feed.Name, feed.Url)
			continue
		}

		err = notifyGroup(config, feed.Group, newItems)
		if err != nil {
			log.Printf("notify error %s: could not notify group: %s\n", feed.Group, err)
			continue
		}

		err = markItemsAsPosted(dataDir, newItems)
		if err != nil {
			log.Printf("feed %s: could not mark %d items as read", feed.Name, len(newItems))
			continue
		}
	}

	return nil
}

func filterNewItems(dataDir string, feed *gofeed.Feed) []*gofeed.Item {
	return nil
}

func notifyGroup(config *config.Config, groupName string, items []*gofeed.Item) error {
	return nil
}

func markItemsAsPosted(dataDir string, items []*gofeed.Item) error {
	return nil
}
