package reddit_go

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// Thing holds attributes common to all reddit api entities.
//
// See https://github.com/reddit/reddit/wiki/JSON
type Thing struct {
	ID   string      `json:"id"`
	Name string      `json:"name"`
	Kind string      `json:"kind"`
	Data interface{} `json:"data"`
}

type thingJSON struct {
	ID   string          `json:"id"`
	Name string          `json:"name"`
	Kind string          `json:"kind"`
	Data json.RawMessage `json:"data"`
}

func (t *Thing) UnmarshalJSON(b []byte) error {
	var j thingJSON
	if err := json.Unmarshal(b, &j); err != nil {
		return err
	}
	var val interface{}
	switch j.Kind {
	case "Listing":
		val = &Listing{}
	case "t1":
		val = &Comment{}
	case "t2":
		val = &Account{}
	case "t3":
		val = &Link{}
	case "t4":
		val = &Message{}
	case "t5":
		val = &SubReddit{}
	default:
		return fmt.Errorf("unsupported kind: %s", j.Kind)
	}
	if err := json.Unmarshal(j.Data, val); err != nil {
		return err
	}
	t.ID, t.Name, t.Kind, t.Data  = j.ID, j.Name, j.Kind, val
	return nil
}

// Listing contains paginated content from an API request.
//
// See https://github.com/reddit/reddit/wiki/JSON
type Listing struct {
	Before   string  `json:"before"`
	After    string  `json:"after"`
	Modhash  string  `json:"modhash"`
	Children []Thing `json:"children"`
}

// Votable holds attributes related to voting.
//
// See https://github.com/reddit/reddit/wiki/JSON
type Votable struct {
	Ups   int  `json:"ups"`
	Downs int  `json:"downs"`
	Likes bool `json:"likes"`
}

// Created holds creation time information.
//
// See https://github.com/reddit/reddit/wiki/JSON
type Created struct {
	Created    float64 `json:"created"`
	CreatedUTC float64 `json:"created_utc"`
}

type Edited struct {
	Unix   float64
	Edited bool
}

func (e *Edited) UnmarshalJSON(b []byte) error {
	str := string(b)
	if str == "false" {
		e.Unix, e.Edited = 0, false
		return nil
	}
	val, err := strconv.ParseFloat(string(b), 64)
	if err != nil {
		return err
	}
	e.Unix, e.Edited = val, true
	return nil
}

func (e *Edited) MarshalJSON() ([]byte, error) {
	if e.Edited {
		return []byte(fmt.Sprintf("%d", e.Unix)), nil
	}
	return []byte("false"), nil
}

// Comment represents a single reddit comment.
//
// See https://github.com/reddit/reddit/wiki/JSON
type Comment struct {
	Votable
	Created
	ApprovedBy          string  `json:"approved_by"`
	Author              string  `json:"author"`
	AuthorFlairCSSClass string  `json:"author_flair_css_class"`
	AuthorFlairText     string  `json:"author_flair_text"`
	BannedBy            string  `json:"banned_by"`
	Body                string  `json:"body"`
	BodyHTML            string  `json:"body_html"`
	Edited              Edited  `json:"edited"`
	Gilded              int     `json:"gilded"`
	Likes               bool    `json:"likes"`
	LinkAuthor          string  `json:"link_author"`
	LinkID              string  `json:"link_id"`
	LinkTitle           string  `json:"link_title"`
	LinkURL             string  `json:"link_url"`
	NumReports          int     `json:"num_reports"`
	ParentID            string  `json:"parent_id"`
	Replies             []Thing `json:"replies"`
	Saved               bool    `json:"saved"`
	Score               int     `json:"score"`
	ScoreHidden         bool    `json:"score_hidden"`
	Subreddit           string  `json:"subreddit"`
	SubredditID         string  `json:"subreddit_id"`
	Distinguished       string  `json:"distinguished"`
}

// Comment represents a single link on reddit.
//
// See https://github.com/reddit/reddit/wiki/JSON
type Link struct {
	Votable
	Created
	Author              string          `json:"author"`
	AuthorFlairCSSClass string          `json:"author_flair_css_class"`
	AuthorFlairText     string          `json:"author_flair_text"`
	Clicked             bool            `json:"clicked"`
	Domain              string          `json:"domain"`
	Hidden              bool            `json:"hidden"`
	IsSelf              bool            `json:"is_self"`
	Likes               bool            `json:"likes"`
	LinkFlairCSSClass   string          `json:"link_flair_css_class"`
	LinkFlairText       string          `json:"link_flair_text"`
	Locked              bool            `json:"locked"`
	Media               json.RawMessage `json:"media"`
	MediaEmbed          json.RawMessage `json:"media_embed"`
	NumComments         int             `json:"num_comments"`
	Over18              bool            `json:"over_18"`
	Permalink           string          `json:"permalink"`
	Saved               bool            `json:"saved"`
	Score               int             `json:"score"`
	Selftext            string          `json:"selftext"`
	SelftextHTML        string          `json:"selftext_html"`
	Subreddit           string          `json:"subreddit"`
	SubredditID         string          `json:"subreddit_id"`
	Thumbnail           string          `json:"thumbnail"`
	Title               string          `json:"title"`
	URL                 string          `json:"url"`
	Edited              Edited          `json:"edited"`
	Distinguished       string          `json:"distinguished"`
	Stickied            bool            `json:"stickied"`
}

type HeaderSize struct {
	Width  int
	Height int
}

func (h *HeaderSize) UnmarshalJSON(b []byte) error {
	v := []int{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	if len(v) == 0 {
		return nil
	}
	if len(v) != 2 {
		return fmt.Errorf("expected 2 element array, got %d elements (%v)", len(v), v)
	}
	h.Width, h.Height = v[0], v[1]
	return nil
}

func (h *HeaderSize) MarshalJSON() ([]byte, error) {
	if h == nil {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf("[%d, %d]", h.Width, h.Height)), nil
}

// SubReddit represents a single subreddit.
//
// See https://github.com/reddit/reddit/wiki/JSON
type SubReddit struct {
	AccountsActive       int         `json:"accounts_active"`
	CommentScoreHideMins int         `json:"comment_score_hide_mins"`
	Description          string      `json:"description"`
	DescriptionHTML      string      `json:"description_html"`
	DisplayName          string      `json:"display_name"`
	HeaderImg            string      `json:"header_img"`
	HeaderSize           *HeaderSize `json:"header_size"`
	HeaderTitle          string      `json:"header_title"`
	Over18               bool        `json:"over18"`
	PublicDescription    string      `json:"public_description"`
	PublicTraffic        bool        `json:"public_traffic"`
	Subscribers          int64       `json:"subscribers"`
	SubmissionType       string      `json:"submission_type"`
	SubmitLinkLabel      string      `json:"submit_link_label"`
	SubmitTextLabel      string      `json:"submit_text_label"`
	SubredditType        string      `json:"subreddit_type"`
	Title                string      `json:"title"`
	URL                  string      `json:"url"`
	UserIsBanned         bool        `json:"user_is_banned"`
	UserIsContributor    bool        `json:"user_is_contributor"`
	UserIsModerator      bool        `json:"user_is_moderator"`
	UserIsSubscriber     bool        `json:"user_is_subscriber"`
}

// Message represents a single message on reddit.
//
// See https://github.com/reddit/reddit/wiki/JSON
type Message struct {
	Created
	Author           string `json:"author"`
	Body             string `json:"body"`
	BodyHTML         string `json:"body_html"`
	Context          string `json:"context"`
	FirstMessage     string `json:"first_message"`
	FirstMessageName string `json:"first_message_name"`
	Likes            bool   `json:"likes"`
	LinkTitle        string `json:"link_title"`
	Name             string `json:"name"`
	New              bool   `json:"new"`
	ParentID         string `json:"parent_id"`
	Replies          string `json:"replies"`
	Subject          string `json:"subject"`
	Subreddit        string `json:"subreddit"`
	WasComment       bool   `json:"was_comment"`
}

// Account represents a single account on reddit.
//
// See https://github.com/reddit/reddit/wiki/JSON
type Account struct {
	Created
	CommentKarma     int    `json:"comment_karma"`
	HasMail          bool   `json:"has_mail"`
	HasModMail       bool   `json:"has_mod_mail"`
	HasVerifiedEmail bool   `json:"has_verified_email"`
	ID               string `json:"id"`
	InboxCount       int    `json:"inbox_count"`
	IsFriend         bool   `json:"is_friend"`
	IsGold           bool   `json:"is_gold"`
	IsMod            bool   `json:"is_mod"`
	LinkKarma        int    `json:"link_karma"`
	Modhash          string `json:"modhash"`
	Name             string `json:"name"`
	Over18           bool   `json:"over_18"`
}

// More holds a list of Thing IDs that are present but not included in full in a response.
//
// See https://github.com/reddit/reddit/wiki/JSON
type More struct {
	Children []string `json:"children"`
}
