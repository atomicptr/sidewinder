package service

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/atomicptr/sidewinder/pkg/config"
)

type testFeedItem struct {
	Title     string
	Link      string
	Published time.Time
}

func TestTickDoesNotAdvanceCursorOnEmptyPoll(t *testing.T) {
	feedServer, setItems := newFeedServer(t)
	defer feedServer.Close()

	feed := config.Feed{Name: "Test Feed", Url: feedServer.URL, Group: "test"}
	dataDir := t.TempDir()
	initial := time.Date(2024, time.January, 10, 12, 0, 0, 0, time.UTC)

	setItems([]testFeedItem{{
		Title:     "Old post",
		Link:      "https://example.com/old",
		Published: time.Date(2024, time.January, 9, 12, 0, 0, 0, time.UTC),
	}})

	if err := markItemsAsPosted(dataDir, feed, initial); err != nil {
		t.Fatalf("markItemsAsPosted() error = %v", err)
	}

	cfg := &config.Config{
		Groups: []config.Group{{Name: "test"}},
		Feeds:  []config.Feed{feed},
	}

	if err := tick(cfg, dataDir); err != nil {
		t.Fatalf("tick() error = %v", err)
	}

	if got := readPostedTime(t, dataDir, feed); !got.Equal(initial) {
		t.Fatalf("posted time = %v, want %v", got, initial)
	}
}

func TestTickPostsBackdatedItemsAfterEmptyPoll(t *testing.T) {
	feedServer, setItems := newFeedServer(t)
	defer feedServer.Close()

	webhookServer, webhookCalls := newWebhookServer(http.StatusNoContent)
	defer webhookServer.Close()

	feed := config.Feed{Name: "Test Feed", Url: feedServer.URL, Group: "test"}
	dataDir := t.TempDir()
	initial := time.Date(2024, time.January, 10, 12, 0, 0, 0, time.UTC)
	newer := time.Date(2024, time.January, 11, 12, 0, 0, 0, time.UTC)
	newest := time.Date(2024, time.January, 12, 12, 0, 0, 0, time.UTC)

	if err := markItemsAsPosted(dataDir, feed, initial); err != nil {
		t.Fatalf("markItemsAsPosted() error = %v", err)
	}

	cfg := &config.Config{
		Groups: []config.Group{{
			Name: "test",
			Webhooks: []config.Webhook{{
				Type: config.DiscordType,
				Url:  webhookServer.URL,
			}},
		}},
		Feeds: []config.Feed{feed},
	}

	setItems([]testFeedItem{{
		Title:     "Old post",
		Link:      "https://example.com/old",
		Published: time.Date(2024, time.January, 9, 12, 0, 0, 0, time.UTC),
	}})

	if err := tick(cfg, dataDir); err != nil {
		t.Fatalf("first tick() error = %v", err)
	}

	if got := readPostedTime(t, dataDir, feed); !got.Equal(initial) {
		t.Fatalf("posted time after empty poll = %v, want %v", got, initial)
	}

	setItems([]testFeedItem{
		{
			Title:     "New post one",
			Link:      "https://example.com/new-one",
			Published: newer,
		},
		{
			Title:     "New post two",
			Link:      "https://example.com/new-two",
			Published: newest,
		},
	})

	if err := tick(cfg, dataDir); err != nil {
		t.Fatalf("second tick() error = %v", err)
	}

	bodies := webhookCalls.Bodies()
	if len(bodies) != 1 {
		t.Fatalf("webhook call count = %d, want 1", len(bodies))
	}

	if !strings.Contains(bodies[0], "2 new posts") {
		t.Fatalf("webhook body %q does not contain grouped post count", bodies[0])
	}

	if got := readPostedTime(t, dataDir, feed); !got.Equal(newest) {
		t.Fatalf("posted time = %v, want %v", got, newest)
	}
}

func TestTickDoesNotAdvanceCursorWhenBatchWebhookFails(t *testing.T) {
	feedServer, setItems := newFeedServer(t)
	defer feedServer.Close()

	webhookServer, webhookCalls := newWebhookServer(http.StatusInternalServerError)
	defer webhookServer.Close()

	feed := config.Feed{Name: "Test Feed", Url: feedServer.URL, Group: "test"}
	dataDir := t.TempDir()
	initial := time.Date(2024, time.January, 10, 12, 0, 0, 0, time.UTC)

	if err := markItemsAsPosted(dataDir, feed, initial); err != nil {
		t.Fatalf("markItemsAsPosted() error = %v", err)
	}

	setItems([]testFeedItem{
		{
			Title:     "New post one",
			Link:      "https://example.com/new-one",
			Published: time.Date(2024, time.January, 11, 12, 0, 0, 0, time.UTC),
		},
		{
			Title:     "New post two",
			Link:      "https://example.com/new-two",
			Published: time.Date(2024, time.January, 12, 12, 0, 0, 0, time.UTC),
		},
	})

	cfg := &config.Config{
		Groups: []config.Group{{
			Name: "test",
			Webhooks: []config.Webhook{{
				Type: config.DiscordType,
				Url:  webhookServer.URL,
			}},
		}},
		Feeds: []config.Feed{feed},
	}

	if err := tick(cfg, dataDir); err != nil {
		t.Fatalf("tick() error = %v", err)
	}

	bodies := webhookCalls.Bodies()
	if len(bodies) != 1 {
		t.Fatalf("webhook call count = %d, want 1", len(bodies))
	}

	if got := readPostedTime(t, dataDir, feed); !got.Equal(initial) {
		t.Fatalf("posted time = %v, want %v", got, initial)
	}
}

func readPostedTime(t *testing.T, dataDir string, feed config.Feed) time.Time {
	t.Helper()

	data, err := os.ReadFile(feedTimePath(dataDir, feed))
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}

	ts, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		t.Fatalf("strconv.ParseInt() error = %v", err)
	}

	return time.Unix(ts, 0)
}

func newFeedServer(t *testing.T) (*httptest.Server, func([]testFeedItem)) {
	t.Helper()

	var mu sync.Mutex
	currentFeed := rssDocument(nil)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		body := currentFeed
		mu.Unlock()

		w.Header().Set("Content-Type", "application/rss+xml")
		_, err := io.WriteString(w, body)
		if err != nil {
			t.Fatalf("io.WriteString() error = %v", err)
		}
	}))

	setItems := func(items []testFeedItem) {
		mu.Lock()
		defer mu.Unlock()
		currentFeed = rssDocument(items)
	}

	return server, setItems
}

type webhookCallLog struct {
	mu     sync.Mutex
	bodies []string
}

func (l *webhookCallLog) Add(body string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	defer func() {
		l.bodies = append(l.bodies, body)
	}()
}

func (l *webhookCallLog) Bodies() []string {
	l.mu.Lock()
	defer l.mu.Unlock()

	bodies := make([]string, len(l.bodies))
	copy(bodies, l.bodies)
	return bodies
}

func newWebhookServer(status int) (*httptest.Server, *webhookCallLog) {
	callLog := &webhookCallLog{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		callLog.Add(string(body))
		w.WriteHeader(status)
	}))

	return server, callLog
}

func rssDocument(items []testFeedItem) string {
	var builder strings.Builder
	builder.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	builder.WriteString("<rss version=\"2.0\"><channel>")
	builder.WriteString("<title>Test Feed</title>")
	builder.WriteString("<link>https://example.com</link>")
	builder.WriteString("<description>Test Feed</description>")

	for _, item := range items {
		builder.WriteString(fmt.Sprintf(
			"<item><title>%s</title><link>%s</link><description>%s</description><pubDate>%s</pubDate></item>",
			item.Title,
			item.Link,
			item.Title,
			item.Published.Format(time.RFC1123Z),
		))
	}

	builder.WriteString("</channel></rss>")
	return builder.String()
}
