package reddit_test

import (
	"fmt"
	"log"
	"net/http"

	"github.com/sridharv/reddit-go"
)

func ExampleConfig_Stream() {
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
}
