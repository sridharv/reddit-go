package reddit_go

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/user"
	"path/filepath"
	"time"

	"github.com/google/go-querystring/query"
)

type Credentials struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	ClientID     string `json:"clientID"`
	ClientSecret string `json:"client_secret"`
	UserAgent    string `json:"user_agent"`
}

type AuthToken struct {
	Expires int64  `json:"expires"`
	Token   string `json:"token"`
	Type    string `json:"type"`
}

type Config struct {
	Credentials Credentials `json:"credentials"`
	AuthToken   AuthToken   `json:"token"`
}

const (
	RedditAuthURL        = "https://www.reddit.com/api/v1/access_token"
	RedditAPIURL         = "https://oauth.reddit.com"
	defaultConfigFile    = ".reddit_creds"
	UseDefaultConfigFile = ""
)

func getFilename(file string) (string, error) {
	if file != UseDefaultConfigFile {
		return file, nil
	}
	current, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to detect current user: %v", err)
	}
	return filepath.Join(current.HomeDir, defaultConfigFile), nil
}

func loadConfig(file string) (Config, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return Config{}, fmt.Errorf("failed to read contents of %s: %v", file, err)
	}
	var ret Config
	if err := json.Unmarshal(data, &ret); err != nil {
		return ret, fmt.Errorf("failed to unmarshal contents of %s to json: %v", file, err)
	}

	errors := notZero("username", ret.Credentials.Username != "") +
		notZero("password", ret.Credentials.Password != "") +
		notZero("client id", ret.Credentials.ClientID != "") +
		notZero("client secret", ret.Credentials.ClientSecret != "") +
		notZero("user agent", ret.Credentials.UserAgent != "")

	if errors != "" {
		return Config{}, fmt.Errorf("%s", errors)
	}
	return ret, nil
}

func notZero(key string, isNonZero bool) string {
	if isNonZero {
		return ""
	}
	return "No " + key + " present. "
}

func ScriptAuth(configFile string) (Config, error) {
	configFile, err := getFilename(configFile)
	if err != nil {
		return Config{}, err
	}
	cfg, err := loadConfig(configFile)
	if err != nil {
		return Config{}, fmt.Errorf("failed to read creds: %v", err)
	}
	if cfg.AuthToken.Token != "" && time.Unix(cfg.AuthToken.Expires, 0).After(time.Now()) {
		return cfg, nil
	}
	if cfg.AuthToken, err = requestToken(cfg.Credentials); err != nil {
		return Config{}, err
	}
	toStore, err := json.Marshal(cfg)
	if err != nil {
		return Config{}, fmt.Errorf("marshalling config failed: %v", err)
	}
	if err := ioutil.WriteFile(configFile, toStore, 0600); err != nil {
		return Config{}, fmt.Errorf("failed to save auth token to %s: %v", configFile, err)
	}
	return cfg, nil
}

func (c Config) Get(client *http.Client, url string, val interface{}) error {
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

type Stream struct {
	c       Config
	client  *http.Client
	lister  Lister
	listing Listing
	index   int
	err     error
}

func (s *Stream) Error() error { return s.err }

func (s *Stream) indexValid() bool { return s.index >= 0 && s.index < len(s.listing.Children) }

func (s *Stream) Next() bool {
	if s.err != nil {
		fmt.Println("errr!")
		return false
	}
	if s.indexValid() {
		s.index++
	}
	if s.indexValid() {
		// We have cached data
		return true
	}
	if s.listing.Before == "" && s.index != -1 {
		fmt.Println("dry")
		return false
	}
	s.lister.List().After = s.listing.Before
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
	fmt.Println("read", url, len(s.listing.Children))
	return s.indexValid()
}

func (s *Stream) Thing() Thing {
	if s.err == nil && s.indexValid() {
		return s.listing.Children[s.index]
	}
	return Thing{}
}

func (c Config) Stream(client *http.Client, lister Lister) *Stream {
	return &Stream{c: c, client: client, lister: lister, index: -1}
}

type TopDuration string

const (
	TopHour  TopDuration = "hour"
	TopDay   TopDuration = "day"
	TopWeek  TopDuration = "week"
	TopMonth TopDuration = "month"
	TopYear  TopDuration = "year"
	TopAll   TopDuration = "all"
)

type ListingOptions struct {
	After  string `url:"after,omitempty"`
	Before string `url:"before,omitempty"`
	Count  int    `url:"count,omitempty"`
	Limit  int    `url:"limit,omitempty"`
	Show   string `url:"show,omitempty"`
}

type URLer interface {
	URL() (string, error)
}

type Lister interface {
	URLer
	List() *ListingOptions
}

type TopPosts struct {
	ListingOptions
	SubReddit string      `url:"-"`
	Duration  TopDuration `url:"t,omitempty"`
}

func (t *TopPosts) URL() (string, error) {
	v, err := query.Values(t)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/r/%s/top.json?%s", RedditAPIURL, t.SubReddit, v.Encode()), nil
}

func (t *TopPosts) List() *ListingOptions { return &t.ListingOptions }

func httpRequest(req *http.Request, client *http.Client) ([]byte, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request to %v failed: %v", req.URL, err)
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read http response from %v: %v", req.URL, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http error %d for %v: %v", resp.StatusCode, req.URL, string(data))
	}
	return data, nil
}

func requestToken(c Credentials) (AuthToken, error) {
	formData := fmt.Sprintf("grant_type=password&username=%s&password=%s", c.Username, c.Password)
	body := bytes.NewBufferString(formData)

	req, err := http.NewRequest(http.MethodPost, RedditAuthURL, body)
	if err != nil {
		return AuthToken{}, fmt.Errorf("failed to create auth request: %v", err)
	}

	req.Header.Add("User-Agent", c.UserAgent)
	req.SetBasicAuth(c.ClientID, c.ClientSecret)

	authTime := time.Now()
	data, err := httpRequest(req, http.DefaultClient)
	if err != nil {
		return AuthToken{}, err
	}
	d := struct {
		Token     string `json:"access_token"`
		ExpiresIn int64  `json:"expires_in"`
		Type      string `json:"token_type"`
	}{}
	if err := json.Unmarshal(data, &d); err != nil {
		return AuthToken{}, fmt.Errorf("invalid token response: %v: %s", err, string(data))
	}

	errors := notZero("token", d.Token != "") + notZero("expiration", d.ExpiresIn != 0) + notZero("token type", d.Type != "")
	if errors != "" {
		return AuthToken{}, fmt.Errorf("incomplete token response: %s", errors)
	}
	return AuthToken{
		Type: d.Type, Token: d.Token, Expires: authTime.Unix() + d.ExpiresIn,
	}, nil
}