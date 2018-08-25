package sankaku

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	c    *http.Client
	opts *Options
	log  *log.Logger
}

type Options struct {
	APIBaseURL string
	SessionID  string
}

// NewClient creates new client for sankaku
func NewClient(c *http.Client, opts *Options, l *log.Logger) (*Client, error) {
	return &Client{
		c:    c,
		opts: opts,
		log:  l,
	}, nil
}

const (
	userAgent         = "Mozilla/5.0 (Windows NT 6.3; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/36.0.1985.125 Safari/537.73"
	sessionCookieName = "_sankakucomplex_session"
	searchLimit       = 100
)

func (c *Client) logF(format string, a ...interface{}) {
	if c.log != nil {
		c.log.Printf(format, a)
	}
}

func newCookie(name, value string) *http.Cookie {
	expires := time.Now().AddDate(1, 0, 0)
	return &http.Cookie{Name: name, Value: value, Expires: expires, HttpOnly: true}
}

func (c *Client) newRequest(ctx context.Context, method, spath string, body io.Reader) (*http.Request, error) {
	u := fmt.Sprintf("%s%s", c.opts.APIBaseURL, spath)

	req, err := http.NewRequest(method, u, body)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)
	req.Header.Set("User-Agent", userAgent)
	req.AddCookie(newCookie(sessionCookieName, c.opts.SessionID))

	return req, nil
}

type Post struct {
	ID         int    `json:"id"`
	MD5        string `json:"md5"`
	Rating     string `json:"rating"`
	FileURL    string `json:"file_url"`
	PreviewURL string `json:"preview_url"`
	Source     string `json:"source"`
	Tags       []*Tag `json:"tags"`
}

type Tag struct {
	ID           int    `json:"id"`
	Count        int    `json:"count"`
	Type         int    `json:"type"`
	Name         string `json:"name"`
	JapaneseName string `json:"name_ja"`
}

func (c *Client) getJSON(req *http.Request, dest interface{}) error {
	resp, err := c.c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	return dec.Decode(dest)
}

func (c *Client) SearchPost(ctx context.Context, keyword string, page int) ([]*Post, error) {
	spath := fmt.Sprintf("/post/index.json?page=%d&limit=%d&tags=%s", page, searchLimit, url.QueryEscape(keyword))

	posts := make([]*Post, 0)
	req, err := c.newRequest(ctx, "GET", spath, nil)
	if err != nil {
		return nil, err
	}
	if err := c.getJSON(req, &posts); err != nil {
		return nil, err
	}
	return posts, err
}
