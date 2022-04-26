package config

import (
	"os"
	"regexp/syntax"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	rssfeed "github.com/umputun/feed-master/app/feed"
	ytfdeed "github.com/umputun/feed-master/app/youtube"
)

func TestLoad(t *testing.T) {
	assert.NoError(t, os.Setenv("TELEGRAM_SERVER", "tg_server"))
	assert.NoError(t, os.Setenv("TELEGRAM_TOKEN", "tg_token"))
	assert.NoError(t, os.Setenv("TWI_CONSUMER_KEY", "tw_key"))
	assert.NoError(t, os.Setenv("TWI_CONSUMER_SECRET", "tw_secret"))
	assert.NoError(t, os.Setenv("TWI_ACCESS_TOKEN", "tw_access_token"))
	assert.NoError(t, os.Setenv("TWI_ACCESS_SECRET", "tw_access_secret"))

	r, err := Load("testdata/config.yml")
	require.NoError(t, err)

	r.setDefaults()

	assert.Equal(t, 3, len(r.Feeds), "3 sets")
	assert.Equal(t, 2, len(r.Feeds["first"].Sources), "2 feeds in first")
	assert.Equal(t, 1, len(r.Feeds["second"].Sources), "1 feed in second")
	assert.Equal(t, "https://bbb.com/u1", r.Feeds["second"].Sources[0].URL)
	assert.Equal(t, "^filterme*", r.Feeds["filtered"].Filter.Title)
	assert.Equal(t, time.Second*600, r.System.UpdateInterval)
	assert.Equal(t, []ytfdeed.FeedInfo{{Name: "name1", ID: "id1", Type: "playlist"},
		{Name: "name2", ID: "id2", Type: "channel", Language: "ru-ru"}},
		r.YouTube.Channels, "2 yt")
	assert.Equal(t, "yt-dlp --extract-audio --audio-format=mp3 -f m4a/bestaudio \"https://www.youtube.com/watch?v={{.ID}}\" --no-progress -o {{.Filename}}.tmp", r.YouTube.DlTemplate)
	assert.Equal(t, "https://www.youtube.com/videos.xml?channel_id=", r.YouTube.BaseChanURL)
	assert.Equal(t, "https://www.youtube.com/videos.xml?playlist_id=", r.YouTube.BasePlaylistURL)
	assert.Equal(t, "./var/rss", r.YouTube.RSSLocation)

	assert.Equal(t, "Feed Master", r.Feeds["first"].Author)
	assert.Equal(t, "author 2", r.Feeds["second"].Author)

	telegram := r.System.Notifications.Telegram
	assert.Equal(t, "tg_server", telegram.Server)
	assert.Equal(t, "tg_token", telegram.Token)
	assert.Equal(t, time.Minute*5, telegram.Timeout)
	twitter := r.System.Notifications.Twitter
	assert.Equal(t, "tw_key", twitter.ConsumerKey)
	assert.Equal(t, "tw_secret", twitter.ConsumerSecret)
	assert.Equal(t, "tw_access_token", twitter.AccessToken)
	assert.Equal(t, "tw_access_secret", twitter.AccessSecret)
	assert.Equal(t, "{{.Title}}", twitter.Template)
	assert.Equal(t, "nobody@feed-master.com", r.Feeds["first"].OwnerEmail)
	assert.Equal(t, "blah@example.com", r.Feeds["second"].OwnerEmail)
}

func TestLoadConfigNotFoundFile(t *testing.T) {
	r, err := Load("/tmp/29e28b3c-e1a4-4269-a10b-3e9a89a08d45.txt")

	assert.Nil(t, r)
	assert.EqualError(t, err, "open /tmp/29e28b3c-e1a4-4269-a10b-3e9a89a08d45.txt: no such file or directory")
}

func TestLoadConfigInvalidYaml(t *testing.T) {
	r, err := Load("testdata/file.txt")

	assert.Nil(t, r)
	assert.EqualError(t, err, "yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `Not Yaml` into config.Conf")
}

func TestSetDefault(t *testing.T) {
	c := Conf{}
	c.setDefaults()

	assert.Equal(t, time.Minute*5, c.System.UpdateInterval)
	assert.Equal(t, 5, c.System.MaxItems)
	assert.Equal(t, 100, c.System.MaxTotal)
	assert.Equal(t, 5000, c.System.MaxKeepInDB)
	assert.Equal(t, 8, c.System.Concurrent)
	assert.Equal(t, "var/feed-master.bdb", c.System.DB)
	assert.Equal(t, time.Minute*5, c.YouTube.UpdateInterval)
	assert.Equal(t, "/yt/media", c.YouTube.BaseURL)
	assert.Equal(t, "var/yt", c.YouTube.FilesLocation)
	assert.Equal(t, "var/rss", c.YouTube.RSSLocation)
	assert.Equal(t, "yt-dlp --extract-audio --audio-format=mp3 --audio-quality=0 -f m4a/bestaudio \"https://www.youtube.com/watch?v={{.ID}}\" --no-progress -o {{.FileName}}.tmp", c.YouTube.DlTemplate)
	assert.Equal(t, "https://www.youtube.com/feeds/videos.xml?channel_id=", c.YouTube.BaseChanURL)
	assert.Equal(t, "https://www.youtube.com/feeds/videos.xml?playlist_id=", c.YouTube.BasePlaylistURL)
	assert.Equal(t, "https://api.telegram.org", c.System.Notifications.Telegram.Server)
	assert.Equal(t, time.Minute*1, c.System.Notifications.Telegram.Timeout)
	assert.Equal(t, "{{.Title}} - {{.Link}}", c.System.Notifications.Twitter.Template)
}

func TestFilterAllCases(t *testing.T) {
	tbl := []struct {
		filter Filter
		inp    rssfeed.Item
		err    error
		out    bool
	}{
		{
			Filter{Title: "(Part \\d+)"},
			rssfeed.Item{Title: "Title (Part 1)"},
			nil,
			true,
		},
		{
			Filter{},
			rssfeed.Item{Title: "Title"},
			nil,
			false,
		},
		{
			Filter{Title: "("},
			rssfeed.Item{Title: "Title"},
			&syntax.Error{Code: "missing closing )", Expr: "("},
			false,
		},
	}

	for i, tb := range tbl {
		tb := tb
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			result, err := tb.filter.Skip(tb.inp)
			assert.Equal(t, tb.out, result)
			assert.Equal(t, tb.err, err)
		})
	}
}
