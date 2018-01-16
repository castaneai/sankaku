package sankaku

import (
	"context"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

type Client struct {
	c    *http.Client
	opts *Options
	log  *log.Logger
}

type Options struct {
	Host      string
	Lang      string
	SessionID string
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
	staticImageHost   = "https://cs.sankakucomplex.com"
)

func (c *Client) logF(format string, a ...interface{}) {
	if c.log != nil {
		c.log.Printf(format, a)
	}
}

func (c *Client) newRequest(ctx context.Context, method, spath string, body io.Reader) (*http.Request, error) {
	spath = strings.TrimLeft(spath, "/")
	u := fmt.Sprintf("%s/%s/%s", c.opts.Host, c.opts.Lang, spath)
	c.logF("start new request: %s", u)

	req, err := http.NewRequest(method, u, body)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)
	req.Header.Set("User-Agent", userAgent)

	expires := time.Now().AddDate(1, 0, 0)
	cookie := http.Cookie{Name: sessionCookieName, Value: c.opts.SessionID, Expires: expires, HttpOnly: true}
	req.AddCookie(&cookie)

	return req, nil
}

func (c *Client) getGoQueryDoc(ctx context.Context, spath string) (*goquery.Document, error) {
	req, err := c.newRequest(ctx, "GET", spath, nil)
	if err != nil {
		return nil, err
	}

	res, err := c.c.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[sankaku] %s", res.Status)
	}

	return goquery.NewDocumentFromResponse(res)
}

type PostInfo struct {
	ID           string   `json:"id"`
	Tags         []string `json:"tags"`
	ThumbnailURL string   `json:"thumbnail_url"`
}

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
			ThumbnailURL: getFullURL(s.Find(".preview").AttrOr("src", "")),
		})
	})
	return posts, nil
}

type Post struct {
	ID           string   `json:"id"`
	URL          string   `json:"url"`
	Tags         []string `json:"tags"`
	ThumbnailURL string   `json:"thumbnail_url"`
	Hash         string   `json:"hash"`
	ResizedURL   string   `json:"resized_url"`
	OriginalURL  string   `json:"original_url"`
	SourceURL    string   `json:"source_url"`
}

func getFullURL(s string) string {
	if strings.HasPrefix(s, "//") {
		return fmt.Sprintf("https:%s", s)
	}
	return s
}

func getThumbnailURL(postHash string) string {
	return fmt.Sprintf("%s/data/preview/%s/%s/%s.jpg", staticImageHost, postHash[0:2], postHash[2:4], postHash)
}

func (c *Client) getPostPath(postID string) string {
	return fmt.Sprintf("/post/show/%s", postID)
}

func (c *Client) getPostURL(postID string) string {
	return fmt.Sprintf("%s/%s%s", c.opts.Host, c.opts.Lang, c.getPostPath(postID))
}

func (c *Client) GetPost(ctx context.Context, postID string) (*Post, error) {
	spath := c.getPostPath(postID)
	gq, err := c.getGoQueryDoc(ctx, spath)
	if err != nil {
		return nil, err
	}

	// TODO: error handling
	post := &Post{ID: postID, URL: c.getPostURL(postID)}
	gq.Find("#stats li").Each(func(i int, s *goquery.Selection) {
		if strings.HasPrefix(s.Text(), "Resized") {
			post.ResizedURL = getFullURL(s.Find("a").AttrOr("href", ""))
		} else if strings.HasPrefix(s.Text(), "Original") {
			post.OriginalURL = getFullURL(s.Find("a").AttrOr("href", ""))
		} else if strings.HasPrefix(s.Text(), "Source") {
			post.SourceURL = getFullURL(s.Find("a").AttrOr("href", ""))
		}
	})
	// もうちょっといいとり方ないものか
	if post.OriginalURL == "" {
		return nil, fmt.Errorf("failed to get post OriginalURL (postID: %s)", postID)
	}
	post.Hash = strings.Split(path.Base(post.OriginalURL), ".")[0]

	gq.Find("#tag-sidebar li > a").Each(func(i int, s *goquery.Selection) {
		tag := strings.Replace(s.Text(), " ", "_", -1)
		if tag != "" {
			post.Tags = append(post.Tags, tag)
		}
	})
	if post.Hash != "" {
		post.ThumbnailURL = getThumbnailURL(post.Hash)
	}

	return post, nil
}
