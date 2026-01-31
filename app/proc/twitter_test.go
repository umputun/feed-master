package proc

import (
	"net/url"
	"strconv"
	"testing"

	"github.com/ChimeraCoder/anaconda"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/feed-master/app/feed"
	"github.com/umputun/feed-master/app/proc/mocks"
)

func TestNewTwitterClient(t *testing.T) {
	twiAuth := TwitterAuth{
		ConsumerKey:    "a",
		ConsumerSecret: "b",
		AccessToken:    "c",
		AccessSecret:   "d",
	}

	client := NewTwitterClient(twiAuth, func(feed.Item) string {
		return ""
	}, nil)

	assert.Equal(t, twiAuth, client.TwitterAuth)
}

func TestTwitterSendIfFieldsTwitterAuthEmpty(t *testing.T) {
	cases := []struct {
		consumerKey, consumerSecret, accessToken, accessSecret string
	}{
		{"a", "", "", ""},
		{"", "b", "", ""},
		{"", "", "c", ""},
		{"", "", "", "d"},
	}

	for i, tt := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			twiAuth := TwitterAuth{
				ConsumerKey:    tt.consumerKey,
				ConsumerSecret: tt.consumerSecret,
				AccessToken:    tt.accessToken,
				AccessSecret:   tt.accessSecret,
			}

			twitterFmtFn := func(feed.Item) string {
				return ""
			}

			client := NewTwitterClient(twiAuth, twitterFmtFn, nil)

			assert.NoError(t, client.Send(feed.Item{}))
		})
	}
}

func TestCleanText(t *testing.T) {
	tbl := []struct {
		inp, out string
		max      int
	}{
		{"test", "test", 10},
		{"test 12345 aaaa", "test ...", 10},
		{"<b>test 12345 aaaa</b>", "test ...", 10},
		{"<b>test12345 aaaa</b>", "test12 ...", 10},
	}

	for i, tt := range tbl {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			out := CleanText(tt.inp, tt.max)
			assert.Equal(t, tt.out, out)
		})
	}
}

func TestTwitterSend(t *testing.T) {
	twitPoster := &mocks.TweetPosterMock{PostTweetFunc: func(string, url.Values) (anaconda.Tweet, error) {
		return anaconda.Tweet{}, nil
	}}
	formatter := func(feed.Item) string {
		return "formatted text"
	}

	tClient := NewTwitterClient(TwitterAuth{
		ConsumerKey:    "a",
		ConsumerSecret: "b",
		AccessToken:    "c",
		AccessSecret:   "d",
	}, formatter, twitPoster)

	require.NoError(t, tClient.Send(feed.Item{}))

	require.Len(t, twitPoster.PostTweetCalls(), 1)
	assert.Equal(t, "formatted text", twitPoster.PostTweetCalls()[0].Msg)
	assert.Equal(t, url.Values{"tweet_mode": []string{"extended"}}, twitPoster.PostTweetCalls()[0].V)
}
