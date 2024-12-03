package service

import (
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/atomicptr/sidewinder/pkg/config"

	"github.com/mmcdole/gofeed"
)

func Run(config *config.Config, dataDir string) error {
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

		newItems, t := filterNewItems(dataDir, feed, f)

		if len(newItems) == 0 {
			log.Printf("feed %s: %s has no new items", feed.Name, feed.Url)
			continue
		}

		err = notifyGroup(config, feed, newItems)
		if err != nil {
			log.Printf("notify error %s: could not notify group: %s\n", feed.Group, err)
			continue
		}

		err = markItemsAsPosted(dataDir, feed, t)
		if err != nil {
			log.Printf("feed %s: could not mark %d items as read: %s", feed.Name, len(newItems), err)
			continue
		}
	}

	return nil
}

func filterNewItems(dataDir string, feed config.Feed, rssFeed *gofeed.Feed) ([]*gofeed.Item, time.Time) {
	var newItems []*gofeed.Item

	t, err := lastItemPostedTime(dataDir, feed)
	if err != nil {
		t = time.Now()
	}

	for _, f := range rssFeed.Items {
		if t.After(*f.PublishedParsed) {
			continue
		}

		log.Printf("%s: found new item: %s - %s\n", feed.Name, f.Title, f.Link)

		newItems = append(newItems, f)
	}

	return newItems, t
}

func notifyGroup(config *config.Config, feed config.Feed, items []*gofeed.Item) error {
	g := config.FindGroup(feed.Group)

	for _, hook := range g.Webhooks {
		for _, item := range items {
			log.Printf("notify group %s about %s - %s\n", feed.Group, item.Title, item.Link)
			err := hook.Fire(feed, item.Title, item.Description, item.Link, *item.PublishedParsed)
			if err != nil {
				log.Printf("notify group %s error: %s\n", g.Name, err)
			}

			// sleep between every request to not get rate limited
			time.Sleep(1 * time.Second)
		}
	}

	return nil
}

func markItemsAsPosted(dataDir string, feed config.Feed, t time.Time) error {
	p := feedTimePath(dataDir, feed)
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	defer func() {
		err := f.Close()
		if err != nil {
			log.Println("could not close file: ", p, err)
		}
	}()

	data := []byte(strconv.FormatInt(t.Unix(), 10))

	log.Printf("marked feed %s as posted for %s\n", p, string(data))

	_, err = f.Write(data)
	return err
}

func feedTimePath(dataDir string, feed config.Feed) string {
	h := md5.New()
	io.WriteString(h, feed.Url)
	ident := fmt.Sprintf("%x", h.Sum(nil))

	return filepath.Join(dataDir, ident)
}

func lastItemPostedTime(dataDir string, feed config.Feed) (time.Time, error) {
	p := feedTimePath(dataDir, feed)

	if _, err := os.Stat(p); os.IsNotExist(err) {
		log.Printf("feed item: %s does not exist\n", p)
		return time.Time{}, err // item does not exist so latest was now
	}

	data, err := os.ReadFile(p)
	if err != nil {
		log.Printf("feed item: %s could not be read\n", p)
		return time.Time{}, err // cant read file?
	}

	ts, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		log.Printf("feed item: %s contains invalid data\n", p)
		return time.Time{}, err
	}

	return time.Unix(ts, 0), nil
}
