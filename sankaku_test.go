package sankaku

import (
	"context"
	"os"
	"testing"
	"time"
	"net/http"
)

func newTestClient() (*Client, error) {
	sessionID := os.Getenv("SANKAKU_SESSION")
	opts := &Options{Host: "https://chan.sankakucomplex.com", Lang: "en", SessionID: sessionID}
	hc := &http.Client{}
	return NewClient(hc, opts, nil)
}

func TestSearchPosts(t *testing.T) {
	c, err := newTestClient()
	if err != nil {
		t.Error(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	posts, err := c.SearchPosts(ctx, "rating:s", 1)
	if err != nil {
		t.Error(err)
	}
	for _, p := range posts {
		t.Logf("%s: %v", p.ID, p.Tags)
	}
}

func TestGetPost(t *testing.T) {
	c, err := newTestClient()
	if err != nil {
		t.Error(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	post, err := c.GetPost(ctx, "6397602")
	if err != nil {
		t.Error(err)
	}
	t.Logf("%v", post)
}
