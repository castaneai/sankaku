package sankaku

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"
)

const (
	apiBaseURL     = "https://capi-beta.sankakucomplex.com"
	defaultTimeout = 60 * time.Second
)

func newTestClient() (*Client, error) {
	sessionID := os.Getenv("SANKAKU_SESSION")
	opts := &Options{APIBaseURL: apiBaseURL, SessionID: sessionID}
	hc := &http.Client{}
	return NewClient(hc, opts, nil)
}

func TestSearchPosts(t *testing.T) {
	c, err := newTestClient()
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	posts, err := c.SearchPost(ctx, "rating:s", 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range posts {
		t.Logf("%+v", p)
	}
}
