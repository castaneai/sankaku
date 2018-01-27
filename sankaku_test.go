package sankaku

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"
)

const (
	Timeout = 60 * time.Second
)

func newTestClient(lang string) (*Client, error) {
	sessionID := os.Getenv("SANKAKU_SESSION")
	opts := &Options{Host: "https://chan.sankakucomplex.com", Lang: lang, SessionID: sessionID}
	hc := &http.Client{}
	return NewClient(hc, opts, nil)
}

func TestSearchPosts(t *testing.T) {
	for _, lang := range []string{"en", "ja"} {
		c, err := newTestClient(lang)
		if err != nil {
			t.Error(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), Timeout)
		defer cancel()
		posts, err := c.SearchPostInfos(ctx, "rating:s", 1)
		if err != nil {
			t.Error(err)
		}
		for _, p := range posts {
			t.Logf("%s: %+v", p.ID, p.Tags)
		}
	}
}

func TestGetPost(t *testing.T) {
	for _, lang := range []string{"en", "ja"} {
		c, err := newTestClient(lang)
		if err != nil {
			t.Error(err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), Timeout)
		defer cancel()

		post, err := c.GetPost(ctx, "6397602")
		if err != nil {
			t.Error(err)
		}
		t.Logf("%+v", post)
	}
}
