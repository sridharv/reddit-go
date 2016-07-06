package reddit_go

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"bytes"

	"encoding/base64"

	"time"

	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/require"
)

type response struct {
	requestURL string
	headers    map[string]string
	body       string
	statusCode int
	response   string
	err        string
}

type mocks struct {
	time      time.Time
	orig      doer
	origClock clockwork.Clock
	ctr       int
	expected  []response
}

func (f *mocks) do(req *http.Request, client *http.Client) (*http.Response, error) {
	if f.ctr >= len(f.expected) {
		return nil, fmt.Errorf("unexpected request received: all responses finished (%d present)", len(f.expected))
	}
	r := f.expected[f.ctr]
	if req.URL.String() != r.requestURL {
		return nil, fmt.Errorf("expected URL: %v, got %s", r.requestURL, req.URL)
	}
	if r.err != "" {
		return nil, fmt.Errorf("%s", r.err)
	}
	for k, v := range r.headers {
		actual := req.Header.Get(k)
		if actual != v {
			return nil, fmt.Errorf("expected value %s for header %s, got %s", v, k, actual)
		}
	}

	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	d := string(data)
	if d != r.body {
		return nil, fmt.Errorf("expected body %s, got %s", r.body, d)
	}

	return &http.Response{
		StatusCode: r.statusCode,
		Body:       ioutil.NopCloser(bytes.NewBufferString(r.response)),
	}, nil
}

func (d *mocks) reset() {
	defaultDoer = d.orig
}

func mock(r ...response) *mocks {
	d := &mocks{orig: defaultDoer, expected: r, origClock: clock, time: time.Now()}
	clock = clockwork.NewFakeClockAt(d.time)
	defaultDoer = d
	return d
}

var testConfig = Config{
	Credentials: Credentials{
		Username:     "blah",
		Password:     "pass",
		ClientID:     "client",
		ClientSecret: "secret",
		UserAgent:    "useragent",
	},
}

const (
	testConfigStr = `{
	"credentials": {
		"username": "blah",
		"password": "pass",
		"clientID": "client",
		"client_secret": "secret",
		"user_agent": "useragent"
	}
}`

	testTokenResponse = `{
	"access_token": "test-token",
    "token_type": "bearer",
    "expires_in": 3600,
    "scope": "*"
}`
)

func TestConfigLoadAndSave(t *testing.T) {
	require := require.New(t)
	tmpDir, err := ioutil.TempDir("", "test")
	require.NoError(err)

	defer os.RemoveAll(tmpDir)
	file := filepath.Join(tmpDir, "creds_file")
	require.NoError(ioutil.WriteFile(file, []byte(testConfigStr), 0600))

	c, err := LoadConfig(file)
	require.NoError(err)

	require.Equal(testConfig, *c)

	c.AuthToken = AuthToken{
		Token: "token",
	}
	require.NoError(c.Save(file))

	c2, err := LoadConfig(file)
	require.NoError(err)
	require.Equal(*c, *c2)
}

func TestScriptAuth(t *testing.T) {
	m := mock(
		response{
			statusCode: 200,
			requestURL: RedditAuthURL,
			response:   testTokenResponse,
			headers: map[string]string{
				"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte("client:secret")),
			},
			body: "grant_type=password&username=blah&password=pass",
		},
	)
	defer m.reset()

	require := require.New(t)

	cfgVal := testConfig
	c := &cfgVal
	require.NoError(c.ScriptAuth(nil))
	require.Equal(AuthToken{Token: "test-token", Type: "bearer", Expires: m.time.Add(time.Hour).Unix()}, c.AuthToken)
}

//func sTestGeneric(t *testing.T) {
//	cfg, err := ScriptAuth(DefaultConfigFile, http.DefaultClient)
//	if err != nil {
//		t.Fatal(err)
//	}
//	req := &TopPosts{SubReddit: "programming", Duration: TopDay, ListingOptions: ListingOptions{Limit: 50}}
//	stream := cfg.Stream(http.DefaultClient, req)
//	for ctr := 0; ctr < 1500 && stream.Next(); ctr++ {
//		//thing := stream.Thing()
//		//l := thing.Data.(*Link)
//		//fmt.Println(l.URL)
//	}
//	if err := stream.Error(); err != nil {
//		t.Fatal(err)
//	}
//}
