package proc

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/ChimeraCoder/anaconda"
	"github.com/denisbrodbeck/striphtmltags"
	log "github.com/go-pkgz/lgr"

	"github.com/umputun/feed-master/app/feed"
)

//go:generate moq -out mocks/tweet_poster.go -pkg mocks -skip-ensure -fmt goimports . TweetPoster

// TweetPoster is the interface for posting Tweets to Twitter
type TweetPoster interface {
	PostTweet(msg string, v url.Values) (tweet anaconda.Tweet, err error)
}

// TwitterClient implements basic publisher of rss item to twitter
type TwitterClient struct {
	TwitterAuth
	formatter   func(feed.Item) string
	tweetPoster TweetPoster
}

// TwitterAuth contains keys and secrets for twitter API
type TwitterAuth struct {
	ConsumerKey, ConsumerSecret string
	AccessToken, AccessSecret   string
}

// NewTwitterClient makes twitter notifier
func NewTwitterClient(auth TwitterAuth, formatter func(feed.Item) string, twitterSender TweetPoster) *TwitterClient {
	return &TwitterClient{TwitterAuth: auth, formatter: formatter, tweetPoster: twitterSender}
}

// Send formatted item to twitter
func (t *TwitterClient) Send(item feed.Item) error {
	if t.ConsumerKey == "" || t.ConsumerSecret == "" || t.AccessToken == "" || t.AccessSecret == "" {
		return nil
	}

	log.Printf("[INFO] publish to twitter %+v", item.Title)

	v := url.Values{}
	v.Set("tweet_mode", "extended")
	msg := t.formatter(item)
	if _, err := t.tweetPoster.PostTweet(msg, v); err != nil {
		return fmt.Errorf("can't send to twitter: %w", err)
	}
	log.Printf("[DEBUG] published to twitter %s", strings.ReplaceAll(msg, "\n", " "))
	return nil
}

// CleanText removes html tags and shrinks result
func CleanText(inp string, maximum int) string {
	res := striphtmltags.StripTags(inp)
	if len([]rune(res)) > maximum {
		// 4 symbols reserved for space and three dots on the end
		snippet := []rune(res)[:maximum-4]
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
