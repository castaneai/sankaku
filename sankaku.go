package sankaku

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
)

type Client struct {
	hc   *http.Client
	opts []ClientOption
}

type ClientOption interface {
	Apply(c *Client) error
}

type authOpt struct {
	token string
}

type authTransport struct {
	base  http.RoundTripper
	token string
}

func (a *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("authorization", "Bearer "+a.token)
	return a.base.RoundTrip(req)
}

func (a *authOpt) Apply(c *Client) error {
	c.hc.Transport = &authTransport{base: c.hc.Transport, token: a.token}
	return nil
}

func WithAuthentication(token string) ClientOption {
	return &authOpt{token: token}
}

// NewClient creates new client for sankaku
func NewClient(c *http.Client, opts ...ClientOption) (*Client, error) {
	return &Client{
		hc:   c,
		opts: opts,
	}, nil
}

const (
	apiBaseURL  = "https://capi-v2.sankakucomplex.com"
	userAgent   = "Mozilla/5.0 (Windows NT 6.3; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/36.0.1985.125 Safari/537.73"
	searchLimit = 100
)

func (c *Client) newRequest(ctx context.Context, method, spath string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, apiBaseURL+spath, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	req.Header.Set("user-agent", userAgent)
	return req, nil
}

type Rating string

func (r Rating) String() string {
	switch r {
	case RatingSafe:
		return "safe"
	case RatingQuestionable:
		return "questionable"
	case RatingExplicit:
		return "explicit"
	default:
		return "unknown"
	}
}

const (
	RatingSafe         Rating = "s"
	RatingQuestionable Rating = "q"
	RatingExplicit     Rating = "e"
)

type Post struct {
	ID         int    `json:"id"`
	MD5        string `json:"md5"`
	Rating     Rating `json:"rating"`
	FileURL    string `json:"file_url"`
	PreviewURL string `json:"preview_url"`
	Source     string `json:"source"`
	Tags       []Tag  `json:"tags"`
}

type Tag struct {
	ID           int    `json:"id"`
	Count        int    `json:"count"`
	Type         int    `json:"type"`
	Name         string `json:"name"`
	JapaneseName string `json:"name_ja"`
}

func (c *Client) getJSON(req *http.Request, dest interface{}) error {
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("http error: %v (body: unknown)", resp.Status)
		}
		return fmt.Errorf("http error: %v, %s", resp.Status, string(errBody))
	}

	dec := json.NewDecoder(resp.Body)
	return dec.Decode(dest)
}

func (c *Client) SearchPost(ctx context.Context, keyword string, page int) ([]*Post, error) {
	spath := "/posts"
	params := make(url.Values)
	params.Set("page", fmt.Sprintf("%d", page))
	params.Set("limit", fmt.Sprintf("%d", searchLimit))
	params.Set("language", "english")
	params.Set("tags", keyword)
	if len(params) > 0 {
		spath += "?" + params.Encode()
	}

	var posts []*Post
	req, err := c.newRequest(ctx, "GET", spath, nil)
	if err != nil {
		return nil, err
	}
	if err := c.getJSON(req, &posts); err != nil {
		return nil, err
	}
	return posts, err
}
