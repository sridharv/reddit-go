package reddit_go

import (
	"testing"
	"net/http"
	"fmt"
)

func TestGeneric(t *testing.T) {
	cfg, err := ScriptAuth(UseDefaultConfigFile)
	if err != nil {
		t.Fatal(err)
	}
	req := &TopPosts{SubReddit: "programming", Duration: TopDay, ListingOptions: ListingOptions{Limit: 50}}
	stream := cfg.Stream(http.DefaultClient, req)
	for ctr := 0; ctr < 150 && stream.Next(); ctr++ {
		thing := stream.Thing()
		l := thing.Data.(*Link)
		fmt.Println(l.URL)
	}
	if err := stream.Error(); err != nil {
		t.Fatal(err)
	}
}
