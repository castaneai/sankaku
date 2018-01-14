package sankaku

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestRequest(t *testing.T) {
	sessionID := os.Getenv("SANKAKU_SESSION")
	c, err := NewClient("https://chan.sankakucomplex.com", "en", sessionID, nil)
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

func TestGetPostWithDetail(t *testing.T) {
	sessionID := os.Getenv("SANKAKU_SESSION")
	c, err := NewClient("https://chan.sankakucomplex.com", "en", sessionID, nil)
	if err != nil {
		t.Error(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	post, detail, err := c.GetPostWithDetail(ctx, "6397602")
	if err != nil {
		t.Error(err)
	}
	t.Logf("%v", post)
	t.Logf("%v", detail)
}
