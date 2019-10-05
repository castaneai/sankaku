package sankaku

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"
)

const (
	defaultTimeout = 60 * time.Second
)

func newTestClient() (*Client, error) {
	var opts []ClientOption
	token := os.Getenv("SANKAKU_TOKEN")
	if token != "" {
		opts = append(opts, WithAuthentication(token))
	}
	hc := http.DefaultClient
	return NewClient(hc, opts...)
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
