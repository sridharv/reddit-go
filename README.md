# Minimal Reddit API wrapper for Go.

_This is a work in progress and may contain bugs. It has only been tested for the top posts query._

The [![GoDoc](https://godoc.org/github.com/sridharv/reddit-go?status.svg)](https://godoc.org/github.com/sridharv/reddit-go) contains detailed documentation. The package examples contain details on how to use the package.

Please read the reddit API documentation first before reading these docs.
Some useful links are:

  * https://github.com/reddit/reddit/wiki/API
  * https://github.com/reddit/reddit/wiki/JSON
  * https://github.com/reddit/reddit/wiki/OAuth2-Quick-Start-Example
  * https://github.com/reddit/reddit/wiki/OAuth2
  * https://www.reddit.com/dev/api

This currently only supports OAuth for script apps. It does not support refreshing OAuth tokens. It provides the following:

  * Code to save and load authorization credentials (client id, client secret, etc).
  * A simple API to obtain and store an OAuth token for a script app using these credentials.
  * An API to perform GET requests using the obtained token.
  * An API to stream listings.

## Example Usage

```
cfg, err := reddit.LoadConfig(reddit.DefaultConfigFile)
if err != nil {
    log.Fatal(err)
}
if err := cfg.AuthScript(http.DefaultClient); err != nil {
    log.Fatal(err)
}
// Print top posts of the day from /r/golang
stream := cfg.Stream(http.DefaultClient, &reddit.TopPosts{SubReddit: "golang", Duration: reddit.TopDay})
for stream.Next() {
    thing := stream.Thing()
    link := thing.Data.(*reddit.Link)
    fmt.Println(link.Title, link.URL)
}
if err := stream.Error(); err != nil {
    log.Fatal(err)
}
```

