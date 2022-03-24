package proc

import (
	"net/url"
	"strings"

	"github.com/ChimeraCoder/anaconda"
	"github.com/denisbrodbeck/striphtmltags"
	log "github.com/go-pkgz/lgr"
	"github.com/pkg/errors"

	"github.com/umputun/feed-master/app/feed"
)

// TwitterClient implements basic publisher of rss itom to twitter
type TwitterClient struct {
	TwitterAuth
	formatter func(feed.Item) string
}

// TwitterAuth contains keys and secrets for twitter API
type TwitterAuth struct {
	ConsumerKey, ConsumerSecret string
	AccessToken, AccessSecret   string
}

// NewTwitterClient makes twitter notifier
func NewTwitterClient(auth TwitterAuth, formatter func(feed.Item) string) *TwitterClient {
	return &TwitterClient{TwitterAuth: auth, formatter: formatter}
}

// Send formatted item to twitter
func (t *TwitterClient) Send(item feed.Item) error {
	if t.ConsumerKey == "" || t.ConsumerSecret == "" || t.AccessToken == "" || t.AccessSecret == "" {
		return nil
	}

	log.Printf("[INFO] publish to twitter %+v", item.Title)
	api := anaconda.NewTwitterApiWithCredentials(t.AccessToken, t.AccessSecret, t.ConsumerKey, t.ConsumerSecret)
	v := url.Values{}
	v.Set("tweet_mode", "extended")
	msg := t.formatter(item)
	if _, err := api.PostTweet(msg, v); err != nil {
		return errors.Wrap(err, "can't send to twitter")
	}
	log.Printf("[DEBUG] published to twitter %s", strings.Replace(msg, "\n", " ", -1))
	return nil
}

// CleanText removes html tags and shrinks result, adding 4 symbols on top
func CleanText(inp string, max int) string {
	res := striphtmltags.StripTags(inp)
	if len([]rune(res)) > max {
		snippet := []rune(res)[:max]
		// go back in snippet and found first space
		for i := len(snippet) - 1; i >= 0; i-- {
			if snippet[i] == ' ' {
				snippet = snippet[:i]
				break
			}
		}
		res = string(snippet) + " ..."
	}
	return res
}
