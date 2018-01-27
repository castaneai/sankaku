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
	Lang      string // "en" or "ja"
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
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[sankaku] %s", res.Status)
	}

	return goquery.NewDocumentFromResponse(res)
}

type PostInfo struct {
	ID           string   `json:"id"`
	Tags         []string `json:"tags"`
	ThumbnailURL string   `json:"thumbnail_url"`
	URL          string   `json:"url"`
}

func (c *Client) SearchPostInfos(ctx context.Context, keyword string, page int) ([]*PostInfo, error) {
	spath := fmt.Sprintf("/post/index.content?tags=%s&page=%d", url.QueryEscape(keyword), page)
	doc, err := c.getGoQueryDoc(ctx, spath)
	if err != nil {
		return nil, err
	}

	var pis []*PostInfo
	doc.Find(".thumb").Each(func(i int, s *goquery.Selection) {
		pi := &PostInfo{
			ID:           strings.TrimLeft(s.AttrOr("id", ""), "p"),
			Tags:         strings.Split(s.Find(".preview").AttrOr("title", ""), " "),
			ThumbnailURL: getFullURL(s.Find(".preview").AttrOr("src", "")),
		}
		pi.URL = c.getPostURL(pi.ID)
		pis = append(pis, pi)
	})
	return pis, nil
}

type Post struct {
	ID           string      `json:"id"`
	Tags         []string    `json:"tags"`
	ThumbnailURL string      `json:"thumbnail_url"`
	URL          string      `json:"url"`
	Content      PostContent `json:"content"`
	Source       PostSource  `json:"source"`
}

type PostContent struct {
	Hash string `json:"hash"`
	URL  string `json:"url"`
}

type PostSource struct {
	Title string
	URL   string
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

func (c *Client) getSourceLabelPrefix() string {
	switch c.opts.Lang {
	case "ja":
		return "ソース:"
	default:
		return "Source:"
	}
}

func getPostContentHash(contentURL string) string {
	return strings.Split(path.Base(contentURL), ".")[0]
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
		// get source
		if strings.HasPrefix(s.Text(), c.getSourceLabelPrefix()) {
			source := PostSource{}
			innerLink := s.Find("a")
			if innerLink != nil {
				source.Title = innerLink.Text()
				source.URL = innerLink.AttrOr("href", "")
			} else {
				source.Title = s.Text()
			}
			post.Source = source
		} else {
			// get content
			innerLink := s.Find("a")
			if innerLink != nil {
				linkId := innerLink.AttrOr("id", "")
				if linkId == "highres" {
					url := getFullURL(innerLink.AttrOr("href", ""))
					hash := getPostContentHash(url)
					post.Content = PostContent{URL: url, Hash: hash}
				}
			}
		}
	})
	// post content must be parsed
	if post.Content.URL == "" || post.Content.Hash == "" {
		return nil, fmt.Errorf("could not parse post content (postID: %s)", postID)
	}
	post.ThumbnailURL = getThumbnailURL(post.Content.Hash)

	// parse tags
	gq.Find("#tag-sidebar li > a").Each(func(i int, s *goquery.Selection) {
		tag := strings.Replace(s.Text(), " ", "_", -1)
		if tag != "" {
			post.Tags = append(post.Tags, tag)
		}
	})

	return post, nil
}
