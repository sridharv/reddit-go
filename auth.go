package reddit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"time"

	"github.com/jonboulle/clockwork"
	"github.com/mitchellh/go-homedir"
)

var clock = clockwork.NewRealClock()

// Credentials contains script credentials for a reddit developer account.
type Credentials struct {
	Username     string `json:"username"`	// Reddit username of the developer account.
	Password     string `json:"password"`	// Password for the above user.
	ClientID     string `json:"clientID"`	// Client ID for the script app.
	ClientSecret string `json:"client_secret"` // Client secret for the script app.
	UserAgent    string `json:"user_agent"`	// User Agent to use when making requests.
}

// AuthToken contains an authentication token obtained via OAuth.
type AuthToken struct {
	Expires int64  `json:"expires"` // Expirations time as seconds since the unix epoch
	Token   string `json:"token"`	// OAuth token
	Type    string `json:"type"`	// Type of token (usually just bearer)
}

// Config contains configuration needed to perform reddit API requests. This consists of
// credentials used to obtain a token and the current token. The credentials must be provided
// by the user of this library. Calling AuthScript then authenticates the client and populates
// the AuthToken.
type Config struct {
	Credentials Credentials `json:"credentials"`
	AuthToken   AuthToken   `json:"token"`
}

const (
	// RedditAuthURL is the URL used to obtain an authentication token.
	RedditAuthURL     = "https://www.reddit.com/api/v1/access_token"
	// RedditAPIURL is the base URL used to make API calls.
	RedditAPIURL      = "https://oauth.reddit.com"
	// DefaultConfigFile is the default file used to store API credentials.
	DefaultConfigFile = "~/.reddit_creds"
)

// LoadConfig loads and validates a Config structure stored as JSON from a configuration file.
// If the file path starts with ~ this is expanded to the home directory of the user. All fields
// in the Credentials field must be non-empty.
func LoadConfig(file string) (*Config, error) {
	file, err := homedir.Expand(file)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read contents of %s: %v", file, err)
	}
	var ret Config
	if err := json.Unmarshal(data, &ret); err != nil {
		return nil, fmt.Errorf("failed to unmarshal contents of %s to json: %v", file, err)
	}

	errors := notZero("username", ret.Credentials.Username != "") +
		notZero("password", ret.Credentials.Password != "") +
		notZero("client id", ret.Credentials.ClientID != "") +
		notZero("client secret", ret.Credentials.ClientSecret != "") +
		notZero("user agent", ret.Credentials.UserAgent != "")

	if errors != "" {
		return nil, fmt.Errorf("%s", errors)
	}
	return &ret, nil
}

func notZero(key string, isNonZero bool) string {
	if isNonZero {
		return ""
	}
	return "No " + key + " present. "
}

// Save saves a config in JSON format to the provided file.
// If the file path starts with ~ this is expanded to the home directory of the user. No validation is
// performed on the Config prior to saving.
func (c *Config) Save(file string) error {
	file, err := homedir.Expand(file)
	if err != nil {
		return err
	}
	toStore, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshalling config failed: %v", err)
	}
	if err := ioutil.WriteFile(file, toStore, 0600); err != nil {
		return fmt.Errorf("failed to save auth token to %s: %v", file, err)
	}
	return nil
}

// AuthScript authenticates the client against reddit's API servers using the method described in
// https://github.com/reddit/reddit/wiki/OAuth2-Quick-Start-Example. If Config.AuthToken holds a
// valid token, no authentication is performed.
//
// If authentication is successful Config.AuthToken is populated with the received authentication token.
// Use Config.Save to save this authentication token.
func (c *Config) AuthScript(client *http.Client) error {
	if c.AuthToken.Token != "" && time.Unix(c.AuthToken.Expires, 0).After(clock.Now()) {
		return nil
	}
	token, err := requestToken(c.Credentials, client)
	if err != nil {
		return err
	}
	c.AuthToken = token
	return nil
}

type doer interface {
	do(req *http.Request, client *http.Client) (*http.Response, error)
}

type passthroughDoer struct{}

func (passthroughDoer) do(req *http.Request, client *http.Client) (*http.Response, error) {
	return client.Do(req)
}

var defaultDoer doer = passthroughDoer{}

func httpRequest(req *http.Request, client *http.Client) ([]byte, error) {
	resp, err := defaultDoer.do(req, client)
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

func requestToken(c Credentials, client *http.Client) (AuthToken, error) {
	formData := fmt.Sprintf("grant_type=password&username=%s&password=%s", c.Username, c.Password)
	body := bytes.NewBufferString(formData)

	req, err := http.NewRequest(http.MethodPost, RedditAuthURL, body)
	if err != nil {
		return AuthToken{}, fmt.Errorf("failed to create auth request: %v", err)
	}

	req.Header.Add("User-Agent", c.UserAgent)
	req.SetBasicAuth(c.ClientID, c.ClientSecret)

	authTime := clock.Now()
	data, err := httpRequest(req, client)
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
