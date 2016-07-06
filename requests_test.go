package reddit

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

	"strconv"
	"strings"

	"flag"

	"github.com/jonboulle/clockwork"
	"github.com/serenize/snaker"
	"github.com/stretchr/testify/require"
)

var convert = flag.String("convert", "", "File containing API to generate")

// This is not really a test but more a code generation tool and (ab)uses the test infrastructure.
// To use this, copy and paste the contents of a table from https://github.com/reddit/reddit/wiki/JSON
// into a file. Then run go test -v -run TestConvert --convert=<path-to-file>
// This will generate the internal JSON structure for that type.
// You can then copy+paste the structure into a go file after verifying that things look fine to you.
func TestConvert(t *testing.T) {
	if *convert == "" {
		return
	}
	conversions := map[string]string{
		"object":      "json.RawMessage",
		"list<thing>": "[]Thing",
		"boolean":     "bool",
		"long":        "int64",
	}

	data, err := ioutil.ReadFile(*convert)
	if err != nil {
		t.Fatal(err)
	}

	lines := bytes.Split(data, []byte("\n"))
	for _, line := range lines {
		l := bytes.TrimSpace(line)
		if len(l) == 0 {
			continue
		}
		tokens := bytes.Split(l, []byte("\t"))
		name := snaker.SnakeToCamel(string(bytes.Title(tokens[1])))
		if name == "Id" {
			name = "ID"
		}
		t := string(bytes.ToLower(tokens[0]))
		converted, ok := conversions[t]
		if !ok {
			converted = t
		}
		fmt.Printf("\t%s %s `json:\"%s\"`\n", name, converted, string(tokens[1]))
	}
}

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
	defer func() { f.ctr++ }()
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

	d := ""
	if req.Body != nil {
		data, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		d = string(data)
	}
	if d != r.body {
		return nil, fmt.Errorf("expected body %s, got %s", r.body, d)
	}

	return &http.Response{
		StatusCode: r.statusCode,
		Body:       ioutil.NopCloser(bytes.NewBufferString(r.response)),
	}, nil
}

func (f *mocks) reset() {
	defaultDoer = f.orig
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

var authRequest = response{
	statusCode: 200,
	requestURL: RedditAuthURL,
	response:   testTokenResponse,
	headers: map[string]string{
		"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte("client:secret")),
		"User-Agent":    "useragent",
	},
	body: "grant_type=password&username=blah&password=pass",
}

var requestHeaders = map[string]string{
	"User-Agent":    "useragent",
	"Authorization": "bearer test-token",
}

func TestConfig_ScriptAuth(t *testing.T) {
	m := mock(authRequest)
	defer m.reset()

	require := require.New(t)

	cfgVal := testConfig
	c := &cfgVal
	require.NoError(c.AuthScript(nil))
	require.Equal(AuthToken{Token: "test-token", Type: "bearer", Expires: m.time.Add(time.Hour).Unix()}, c.AuthToken)
}

func topPostsBody(start, count int) string {
	before, after := strconv.Itoa(start), strconv.Itoa(start+count-1)
	if start == 0 {
		before = ""
	}
	if count != 5 {
		after = ""
	}
	children := make([]string, count)
	for i := 0; i < count; i++ {
		children[i] = fmt.Sprintf(`{"kind": "t3", "data": {"author": "author%d" } }`, start+i)
	}
	return fmt.Sprintf(`{
	"kind": "Listing",
	"data": {
		"before": "%s",
		"after": "%s",
		"children": [
		%s
		]
	}
}`, before, after, strings.Join(children, ",\n"))
}

func TestConfig_Stream(t *testing.T) {
	m := mock(
		response{
			statusCode: 200,
			headers:    requestHeaders,
			requestURL: "https://oauth.reddit.com/r/programming/top.json?limit=5&t=day",
			response:   topPostsBody(0, 5),
		},
		response{
			statusCode: 200,
			headers:    requestHeaders,
			requestURL: "https://oauth.reddit.com/r/programming/top.json?after=4&count=5&limit=5&t=day",
			response:   topPostsBody(5, 5),
		},
		response{
			statusCode: 200,
			headers:    requestHeaders,
			requestURL: "https://oauth.reddit.com/r/programming/top.json?after=9&count=10&limit=5&t=day",
			response:   topPostsBody(10, 2),
		},
	)
	defer m.reset()

	require := require.New(t)

	// Assume pre-authed
	c := &Config{
		Credentials: testConfig.Credentials,
		AuthToken:   AuthToken{Token: "test-token", Type: "bearer", Expires: m.time.Add(time.Hour).Unix()},
	}

	req := &TopPosts{SubReddit: "programming", Duration: TopDay, ListingOptions: ListingOptions{Limit: 5}}
	stream, ctr := c.Stream(nil, req), 0
	for ; ctr < 12 && stream.Next(); ctr++ {
		thing := stream.Thing()
		l := thing.Data.(*Link)
		require.Equal(fmt.Sprintf("author%d", ctr), l.Author)
	}
	require.NoError(stream.Error())
	require.False(stream.Next())
	require.Equal(12, ctr)
}
