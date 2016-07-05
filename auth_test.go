package reddit_go

import (
	"net/http"
	"testing"
)

type fakeDoer struct {

}

func TestScriptAuth(t *testing.T) {

}

func TestGeneric(t *testing.T) {
	cfg, err := ScriptAuth(DefaultConfigFile, http.DefaultClient)
	if err != nil {
		t.Fatal(err)
	}
	req := &TopPosts{SubReddit: "programming", Duration: TopDay, ListingOptions: ListingOptions{Limit: 50}}
	stream := cfg.Stream(http.DefaultClient, req)
	for ctr := 0; ctr < 1500 && stream.Next(); ctr++ {
		//thing := stream.Thing()
		//l := thing.Data.(*Link)
		//fmt.Println(l.URL)
	}
	if err := stream.Error(); err != nil {
		t.Fatal(err)
	}
}
