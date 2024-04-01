package config

import (
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
	r, err := Load("testdata/config.yml")
	require.NoError(t, err)

	r.setDefaults()

	assert.Equal(t, 4, len(r.Feeds), "4 sets")
	assert.Equal(t, 2, len(r.Feeds["first"].Sources), "2 feeds in first")
	assert.Equal(t, 1, len(r.Feeds["second"].Sources), "1 feed in second")
	assert.Equal(t, "https://bbb.com/u1", r.Feeds["second"].Sources[0].URL)
	assert.Equal(t, "^filterme*", r.Feeds["filtered"].Filter.Title)
	assert.Equal(t, time.Second*600, r.System.UpdateInterval)
	assert.Equal(t, time.Second*10, r.System.HTTPResponseTimeout)
	assert.Equal(t, []ytfdeed.FeedInfo{{Name: "name1", ID: "id1", Type: "playlist", Keep: 15},
		{Name: "name2", ID: "id2", Type: "channel", Language: "ru-ru", Keep: 5}},
		r.YouTube.Channels, "2 yt")
	assert.Equal(t, "yt-dlp --extract-audio --audio-format=mp3 -f m4a/bestaudio \"https://www.youtube.com/watch?v={{.ID}}\" --no-progress -o {{.Filename}}", r.YouTube.DlTemplate)
	assert.Equal(t, "https://www.youtube.com/videos.xml?channel_id=", r.YouTube.BaseChanURL)
	assert.Equal(t, "https://www.youtube.com/videos.xml?playlist_id=", r.YouTube.BasePlaylistURL)
	assert.Equal(t, "./var/rss", r.YouTube.RSSLocation)

	assert.Equal(t, "Feed Master", r.Feeds["first"].Author)
	assert.Equal(t, "author 2", r.Feeds["second"].Author)

	assert.Equal(t, "nobody@feed-master.com", r.Feeds["first"].OwnerEmail)
	assert.Equal(t, "blah@example.com", r.Feeds["second"].OwnerEmail)

	assert.Equal(t, "(one|two|three)", r.Feeds["filtered2"].Filter.Title)
	assert.Equal(t, true, r.Feeds["filtered2"].Filter.Invert)
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

func TestSingleFeedConf(t *testing.T) {
	cases := []struct {
		feedURL, channel string
		updateInterval   time.Duration
	}{
		{"example.com/feed", "Feed", 10},
		{"example.com/my/feed", "My feed", 20},
	}

	for i, tc := range cases {
		i := i
		tc := tc
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			conf := SingleFeed(tc.feedURL, tc.channel, tc.updateInterval)

			assert.Len(t, conf.Feeds, 1)
			assert.Equal(t, conf.System.UpdateInterval, tc.updateInterval)

			feed := conf.Feeds["auto"]
			assert.Equal(t, feed.TelegramChannel, tc.channel)
			assert.Len(t, feed.Sources, 1)
			assert.Equal(t, feed.Sources[0].Name, "auto")
			assert.Equal(t, feed.Sources[0].URL, tc.feedURL)
		})
	}
}

func TestSetDefault(t *testing.T) {
	c := Conf{}
	c.setDefaults()

	expectedConf := Conf{
		System: struct {
			UpdateInterval      time.Duration `yaml:"update"`
			HTTPResponseTimeout time.Duration `yaml:"http_response_timeout"`
			MaxItems            int           `yaml:"max_per_feed"`
			MaxTotal            int           `yaml:"max_total"`
			MaxKeepInDB         int           `yaml:"max_keep"`
			Concurrent          int           `yaml:"concurrent"`
			BaseURL             string        `yaml:"base_url"`
		}{
			UpdateInterval:      time.Minute * 5,
			HTTPResponseTimeout: time.Second * 30,
			MaxItems:            5,
			MaxTotal:            100,
			MaxKeepInDB:         5000,
			Concurrent:          8,
			BaseURL:             "",
		},
	}

	assert.Equal(t, expectedConf.System, c.System)
	assert.Equal(t, time.Minute*5, c.YouTube.UpdateInterval)
	assert.Equal(t, "/yt/media", c.YouTube.BaseURL)
	assert.Equal(t, "var/yt", c.YouTube.FilesLocation)
	assert.Equal(t, "var/rss", c.YouTube.RSSLocation)
	assert.Equal(t, "yt-dlp --extract-audio --audio-format=mp3 --audio-quality=0 -f m4a/bestaudio \"https://www.youtube.com/watch?v={{.ID}}\" --no-progress -o {{.FileName}} --match-filter \"!is_live & availability=public\"", c.YouTube.DlTemplate)
	assert.Equal(t, "https://www.youtube.com/feeds/videos.xml?channel_id=", c.YouTube.BaseChanURL)
	assert.Equal(t, "https://www.youtube.com/feeds/videos.xml?playlist_id=", c.YouTube.BasePlaylistURL)
}

func TestFilter(t *testing.T) {
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
			Filter{Title: "(Part \\d+)", Invert: true},
			rssfeed.Item{Title: "Title (Part 1)"},
			nil,
			false,
		},
		{
			Filter{Title: "(one|two|three)"},
			rssfeed.Item{Title: "something blah one"},
			nil,
			true,
		},
		{
			Filter{Title: "(one|two|three)", Invert: true},
			rssfeed.Item{Title: "something blah one"},
			nil,
			false,
		},
		{
			Filter{Title: "(one|two|three)", Invert: true},
			rssfeed.Item{Title: "something blah two something"},
			nil,
			false,
		},
		{
			Filter{Title: "(one|two|three)", Invert: true},
			rssfeed.Item{Title: "something blah something"},
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
			Filter{Invert: true},
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
