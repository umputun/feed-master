package proc

import (
	"strconv"
	"testing"

	"github.com/umputun/feed-master/app/feed"

	"github.com/stretchr/testify/assert"
)

func TestNewTwitterClient(t *testing.T) {
	twiAuth := TwitterAuth{
		ConsumerKey:    "a",
		ConsumerSecret: "b",
		AccessToken:    "c",
		AccessSecret:   "d",
	}

	client := NewTwitterClient(twiAuth, func(item feed.Item) string {
		return ""
	})

	assert.EqualValues(t, twiAuth, client.TwitterAuth)
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
		i := i
		tt := tt
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			twiAuth := TwitterAuth{
				ConsumerKey:    tt.consumerKey,
				ConsumerSecret: tt.consumerSecret,
				AccessToken:    tt.accessToken,
				AccessSecret:   tt.accessSecret,
			}

			twitterFmtFn := func(item feed.Item) string {
				return ""
			}

			client := NewTwitterClient(twiAuth, twitterFmtFn)

			assert.Nil(t, client.Send(feed.Item{}))
		})
	}

}

func TestCleanText(t *testing.T) {
	tbl := []struct {
		inp, out string
		max      int
	}{
		{"test", "test", 10},
		{"test 12345 aaaa", "test ...", 6},
		{"<b>test 12345 aaaa</b>", "test ...", 6},
		{"<b>test12345 aaaa</b>", "test12 ...", 6},
	}

	for i, tt := range tbl {
		i := i
		tt := tt
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			out := CleanText(tt.inp, tt.max)
			assert.Equal(t, tt.out, out)
		})
	}
}
