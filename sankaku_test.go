package sankaku

import (
	"context"
	"testing"
	"time"
)

func TestRequest(t *testing.T) {
	c, err := NewClient("https://chan.sankakucomplex.com", "en", nil)
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
