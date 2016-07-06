package reddit

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/go-querystring/query"
)

// Get performs an authentication GET request to the provided URL using the provided http.Client instance.
// Responses are unmarshalled into val.
func (c *Config) Get(client *http.Client, url string, val interface{}) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request for %s: %v", url, err)
	}
	req.Header.Add("User-Agent", c.Credentials.UserAgent)
	req.Header.Add("Authorization", fmt.Sprintf("%s %s", c.AuthToken.Type, c.AuthToken.Token))

	data, err := httpRequest(req, client)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, val); err != nil {
		return fmt.Errorf("failed to parse response from %s: %v", url, err)
	}
	return nil
}

// Stream represents a stream of Thing values obtained from a Listing url.
type Stream struct {
	c       *Config
	client  *http.Client
	lister  Lister
	listing Listing
	index   int
	err     error
}

// Error returns a non-nil error if there were any errors when fetching the listing.
func (s *Stream) Error() error { return s.err }

func (s *Stream) indexValid() bool { return s.index >= 0 && s.index < len(s.listing.Children) }

// Next returns true iff there are more Things to read. It automatically fetches a new Listing when
// the current one is exhausted. Always call Error() after Next returns false to check if any errors
// are present.
func (s *Stream) Next() bool {
	if s.err != nil {
		return false
	}
	if s.indexValid() {
		s.index++
	}
	if s.indexValid() {
		// We have cached data
		return true
	}
	if s.listing.After == "" && s.index != -1 {
		return false
	}
	s.lister.List().After = s.listing.After
	url, err := s.lister.URL()
	if err != nil {
		s.err = err
		return false
	}
	var t Thing
	s.index, s.err = 0, s.c.Get(s.client, url, &t)
	if s.err != nil {
		return false
	}
	s.listing = *(t.Data.(*Listing))
	s.lister.List().Count += len(s.listing.Children)
	return s.indexValid()
}

// Thing returns the current Thing. Call Next to advance to the next Thing in the
// stream. This will return the zero value for Thing if Stream.Error() is non-nil or
// the end of the stream has been reached.
func (s *Stream) Thing() Thing {
	if s.err == nil && s.indexValid() {
		return s.listing.Children[s.index]
	}
	return Thing{}
}

// Stream returns a Stream that pages through a Listing. The provided lister is automatically
// updated to hold the correct After and Count values for paging. All requests are performed using
// the provided http.Client instance.
func (c *Config) Stream(client *http.Client, lister Lister) *Stream {
	return &Stream{c: c, client: client, lister: lister, index: -1}
}

// TopDuration represents a sort value for fetching top posts.
type TopDuration string

// TopHour, TopDay, TopWeek, TopMonth, TopYear and TopAll are supported sort values for a TopPosts request.
const (
	TopHour  TopDuration = "hour"
	TopDay   TopDuration = "day"
	TopWeek  TopDuration = "week"
	TopMonth TopDuration = "month"
	TopYear  TopDuration = "year"
	TopAll   TopDuration = "all"
)

// ListingOptions control the size and position of a streamed listing.
// See https://www.reddit.com/dev/api for more information on what these
// parameters mean.
type ListingOptions struct {
	After  string `url:"after,omitempty"`
	Before string `url:"before,omitempty"`
	Count  int    `url:"count,omitempty"`
	Limit  int    `url:"limit,omitempty"`
	Show   string `url:"show,omitempty"`
}

// URLer returns a URL to be used for an API call.
type URLer interface {
	URL() (string, error)
}

// Lister provides access to a modifiable ListingOptions instance which is used to stream Listings.
type Lister interface {
	URLer
	List() *ListingOptions
}

// TopPosts is a query for the top posts of a specified subreddit. It implements URLer and Lister
// and can be used with Config.Stream to stream the top posts of a subreddit.
type TopPosts struct {
	ListingOptions
	SubReddit string      `url:"-"`
	Duration  TopDuration `url:"t,omitempty"`
}

// URL returns the URL to use when fetching the top posts.
func (t *TopPosts) URL() (string, error) {
	v, err := query.Values(t)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/r/%s/top.json?%s", RedditAPIURL, t.SubReddit, v.Encode()), nil
}

// List returns the ListingOptions for TopPosts
func (t *TopPosts) List() *ListingOptions { return &t.ListingOptions }
