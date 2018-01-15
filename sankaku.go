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
	"path"
)

type Client struct {
	c *http.Client
	opts *Options
	log *log.Logger
}

type Options struct {
	Host string
	Lang string
	SessionID string
}

// NewClient creates new client for sankaku
func NewClient(c *http.Client, opts *Options, l *log.Logger) (*Client, error) {
	if l == nil {
		l = log.New(os.Stdout, "[sankaku]", log.Ltime)
	}

	return &Client{
		c:    c,
		opts: opts,
		log:  l,
	}, nil
}

const (
	userAgent         = "Mozilla/5.0 (Windows NT 6.3; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/36.0.1985.125 Safari/537.73"
	sessionCookieName = "_sankakucomplex_session"
	staticImageHost = "https://cs.sankakucomplex.com"
)

func (c *Client) newRequest(ctx context.Context, method, spath string, body io.Reader) (*http.Request, error) {
	spath = strings.TrimLeft(spath, "/")
	u := fmt.Sprintf("%s/%s/%s", c.opts.Host, c.opts.Lang, spath)
	c.log.Printf("start new request: %s", u)

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

	return goquery.NewDocumentFromResponse(res)
}

type Post struct {
	ID           string
	Tags         []string
	ThumbnailURL string
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

type PostDetail struct {
	ID          string
	Hash        string
	ResizedURL  string
	OriginalURL string
	SourceURL   string
}

func getFullURL(s string) string {
	if strings.HasPrefix(s, "//") {
		return fmt.Sprintf("https:%s", s)
	}
	return s
}

func getThumbnailURL(postHash string) (string) {
	return fmt.Sprintf("%s/data/preview/%s/%s/%s.jpg", staticImageHost, postHash[0:2], postHash[2:4], postHash)
}

func (c *Client) GetPostWithDetail(ctx context.Context, postID string) (*Post, *PostDetail, error) {
	spath := fmt.Sprintf("/post/show/%s", postID)
	doc, err := c.getGoQueryDoc(ctx, spath)
	if err != nil {
		return nil, nil, err
	}

	// TODO: error handling
	detail := &PostDetail{ID: postID}
	doc.Find("#stats li").Each(func(i int, s *goquery.Selection) {
		if strings.HasPrefix(s.Text(), "Resized") {
			detail.ResizedURL = getFullURL(s.Find("a").AttrOr("href", ""))
		} else if strings.HasPrefix(s.Text(), "Original") {
			detail.OriginalURL = getFullURL(s.Find("a").AttrOr("href", ""))
		} else if strings.HasPrefix(s.Text(), "Source") {
			detail.SourceURL = getFullURL(s.Find("a").AttrOr("href", ""))
		}
	})
	// もうちょっといいとり方ないものか
	if detail.OriginalURL == "" {
		return nil, nil, fmt.Errorf("failed to get post OriginalURL (postID: %s)", postID)
	}
	detail.Hash = strings.Split(path.Base(detail.OriginalURL), ".")[0]

	post := &Post{ID: postID}
	doc.Find("#tag-sidebar li > a").Each(func(i int, s *goquery.Selection) {
		tag := strings.Replace(s.Text(), " ", "_", -1)
		if tag != "" {
			post.Tags = append(post.Tags, tag)
		}
	})
	if detail.Hash != "" {
		post.ThumbnailURL = getThumbnailURL(detail.Hash)
	}

	return post, detail, nil
}