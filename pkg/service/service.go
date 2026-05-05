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

		newItems, t, shouldMark, err := filterNewItems(dataDir, feed, f)
		if err != nil {
			log.Printf("feed %s: could not determine last posted time: %s\n", feed.Name, err)
			continue
		}

		if len(newItems) == 0 {
			log.Printf("feed %s: %s has no new items", feed.Name, feed.Url)
		} else {
			err = notifyGroup(config, feed, newItems)
			if err != nil {
				log.Printf("notify error %s: could not notify group: %s\n", feed.Group, err)
				continue
			}
		}

		if !shouldMark {
			continue
		}

		err = markItemsAsPosted(dataDir, feed, t)
		if err != nil {
			log.Printf("feed %s: could not mark items as posted: %s", feed.Name, err)
		}
	}

	return nil
}

func filterNewItems(dataDir string, feed config.Feed, rssFeed *gofeed.Feed) ([]*gofeed.Item, time.Time, bool, error) {
	var newItems []*gofeed.Item

	lastPosted, err := lastItemPostedTime(dataDir, feed)
	if err != nil {
		if os.IsNotExist(err) {
			return newItems, time.Now(), true, nil
		}

		return nil, time.Time{}, false, err
	}

	latestPosted := lastPosted

	for _, f := range rssFeed.Items {
		if f.PublishedParsed == nil {
			log.Printf("%s: skipping item with no published time: %s - %s\n", feed.Name, f.Title, f.Link)
			continue
		}

		if !f.PublishedParsed.After(lastPosted) {
			continue
		}

		log.Printf("%s: found new item: %s - %s\n", feed.Name, f.Title, f.Link)

		newItems = append(newItems, f)

		if f.PublishedParsed.After(latestPosted) {
			latestPosted = *f.PublishedParsed
		}
	}

	if len(newItems) == 0 {
		return nil, time.Time{}, false, nil
	}

	return newItems, latestPosted, true, nil
}

func notifyGroup(cfg *config.Config, feed config.Feed, items []*gofeed.Item) error {
	g := cfg.FindGroup(feed.Group)
	var notifyErr error

	for _, hook := range g.Webhooks {
		if len(items) == 1 {
			item := items[0]
			log.Printf("notify group %s about %s - %s\n", feed.Group, item.Title, item.Link)
			err := hook.Fire(feed, item.Title, item.Description, item.Link, *item.PublishedParsed)
			if err != nil {
				log.Printf("notify group %s error: %s\n", g.Name, err)
				if notifyErr == nil {
					notifyErr = err
				}
			}

			continue
		}

		var batchItems []config.ItemData
		for _, item := range items {
			log.Printf("notify group %s about %s - %s\n", feed.Group, item.Title, item.Link)
			batchItems = append(batchItems, config.ItemData{
				Title:       item.Title,
				Description: item.Description,
				URL:         item.Link,
				Published:   *item.PublishedParsed,
			})
		}

		err := hook.FireBatch(feed, batchItems)
		if err != nil {
			log.Printf("notify group %s error: %s\n", g.Name, err)
			if notifyErr == nil {
				notifyErr = err
			}
		}

		time.Sleep(1 * time.Second)
	}

	return notifyErr
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
