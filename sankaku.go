package sankaku

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// Client ...
type Client struct {
	Host      string
	Lang      string
	SessionID string

	HTTPClient *http.Client
	Logger     *log.Logger
}

// NewClient creates new client for sankaku
func NewClient(host, lang, sessionID string, logger *log.Logger) (*Client, error) {
	if logger == nil {
		logger = log.New(os.Stdout, "[sankaku]", log.Ltime)
	}

	return &Client{
		Host:       host,
		Lang:       lang,
		SessionID:  sessionID,
		HTTPClient: &http.Client{},
		Logger:     logger,
	}, nil
}

const (
	userAgent         = "Mozilla/5.0 (Windows NT 6.3; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/36.0.1985.125 Safari/537.73"
	sessionCookieName = "_sankakucomplex_session"
)

func (c *Client) newRequest(ctx context.Context, method, spath string, body io.Reader) (*http.Request, error) {
	c.Logger.Printf("c.Host: %s", c.Host)
	spath = strings.TrimLeft(spath, "/")
	url := fmt.Sprintf("%s/%s/%s", c.Host, c.Lang, spath)
	c.Logger.Printf("url: %s", url)

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)
	req.Header.Set("User-Agent", userAgent)

	expires := time.Now().AddDate(1, 0, 0)
	cookie := http.Cookie{Name: sessionCookieName, Value: c.SessionID, Expires: expires, HttpOnly: true}
	req.AddCookie(&cookie)

	return req, nil
}

func (c *Client) getGoQueryDoc(ctx context.Context, spath string) (*goquery.Document, error) {
	req, err := c.newRequest(ctx, "GET", spath, nil)
	if err != nil {
		return nil, err
	}

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	return goquery.NewDocumentFromResponse(res)
}

// Post of sankaku
type Post struct {
	ID           string
	Tags         []string
	ThumbnailURL string
}

// SearchPosts ...
func (c *Client) SearchPosts(ctx context.Context, keyword string, page int) ([]Post, error) {
	spath := fmt.Sprintf("/post/index.content?tags=%s&page=%d", url.QueryEscape(keyword), page)
	doc, err := c.getGoQueryDoc(ctx, spath)
	if err != nil {
		return nil, err
	}

	var posts []Post
	doc.Find(".thumb").Each(func(i int, s *goquery.Selection) {
		posts = append(posts, Post{
			ID:           strings.TrimLeft(s.AttrOr("id", ""), "p"),
			Tags:         strings.Split(s.Find(".preview").AttrOr("title", ""), " "),
			ThumbnailURL: "https:" + s.Find(".preview").AttrOr("src", ""),
		})
	})
	return posts, nil
}
