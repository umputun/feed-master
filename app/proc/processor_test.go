package proc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/go-pkgz/lgr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"

	"github.com/umputun/feed-master/app/config"
	"github.com/umputun/feed-master/app/feed"
	"github.com/umputun/feed-master/app/proc/mocks"
	"github.com/umputun/feed-master/app/youtube"
)

func TestProcessor_DoRemoveOldItems(t *testing.T) {
	lgr.Setup(lgr.Debug)
	tgNotif := &mocks.TelegramNotifMock{SendFunc: func(string, feed.Item) error {
		return nil
	}}

	twitterNotif := &mocks.TwitterNotifMock{SendFunc: func(feed.Item) error {
		return nil
	}}

	tmpfile := filepath.Join(os.TempDir(), "test.db")
	defer os.Remove(tmpfile)

	db, err := bolt.Open(tmpfile, 0o600, &bolt.Options{Timeout: 2 * time.Second})
	require.NoError(t, err)
	boltStore := &BoltDB{DB: db}

	testFeed, err := os.ReadFile("./testdata/rss1.xml")
	require.NoError(t, err)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Length", strconv.Itoa(len(testFeed)))
		_, e := w.Write(testFeed)
		require.NoError(t, e)
	}))
	defer ts.Close()

	proc := Processor{
		Conf: &config.Conf{
			Feeds: map[string]config.Feed{
				"feed1": {
					Title:           "title1",
					Description:     "description",
					Link:            "link",
					Image:           "imageUrl",
					Language:        "language",
					TelegramChannel: "tgChannel",
					Filter:          config.Filter{},
					Sources:         []config.Source{{Name: "sourceName", URL: ts.URL}},
					ExtendDateTitle: "",
					Author:          "author",
				},
			},
			System: struct {
				UpdateInterval      time.Duration `yaml:"update"`
				HTTPResponseTimeout time.Duration `yaml:"http_response_timeout"`
				MaxItems            int           `yaml:"max_per_feed"`
				MaxTotal            int           `yaml:"max_total"`
				MaxKeepInDB         int           `yaml:"max_keep"`
				Concurrent          int           `yaml:"concurrent"`
				BaseURL             string        `yaml:"base_url"`
			}{
				UpdateInterval:      time.Second / 2,
				HTTPResponseTimeout: time.Second,
				MaxItems:            5,
				MaxTotal:            5,
				MaxKeepInDB:         5,
				Concurrent:          1,
				BaseURL:             "baseUrl",
			},
			YouTube: struct {
				DlTemplate      string             `yaml:"dl_template"`
				BaseChanURL     string             `yaml:"base_chan_url"`
				BasePlaylistURL string             `yaml:"base_playlist_url"`
				Channels        []youtube.FeedInfo `yaml:"channels"`
				BaseURL         string             `yaml:"base_url"`
				UpdateInterval  time.Duration      `yaml:"update"`
				MaxItems        int                `yaml:"max_per_channel"`
				FilesLocation   string             `yaml:"files_location"`
				RSSLocation     string             `yaml:"rss_location"`
				SkipShorts      time.Duration      `yaml:"skip_shorts"`
				DisableUpdates  bool               `yaml:"disable_updates"`
				YtDlpUpdate     struct {
					Interval time.Duration `yaml:"interval"`
					Command  string        `yaml:"command"`
				} `yaml:"ytdlp_update"`
			}{},
		},
		Store:         boltStore,
		TelegramNotif: tgNotif,
		TwitterNotif:  twitterNotif,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*900)
	defer cancel()

	// running processor first time to load items to db
	err = proc.Do(ctx)
	assert.EqualError(t, err, "context deadline exceeded")

	res, err := boltStore.Load("feed1", 10, false)
	require.NoError(t, err)
	assert.Equal(t, 3, len(res), "all 3 items loaded on the first rss parsing")
	assert.Equal(t, "Радио-Т 798", res[0].Title)
	assert.Equal(t, "Радио-Т 797", res[1].Title)
	assert.Equal(t, "Радио-Т 796", res[2].Title)

	require.Equal(t, 3, len(tgNotif.SendCalls()))
	assert.Equal(t, "Радио-Т 798", tgNotif.SendCalls()[0].Item.Title)
	assert.Equal(t, "tgChannel", tgNotif.SendCalls()[0].ChanID)
	assert.Equal(t, "Радио-Т 797", tgNotif.SendCalls()[1].Item.Title)
	assert.Equal(t, "Радио-Т 796", tgNotif.SendCalls()[2].Item.Title)

	require.Equal(t, 3, len(twitterNotif.SendCalls()))
	assert.Equal(t, "Радио-Т 798", twitterNotif.SendCalls()[0].Item.Title)
	assert.Equal(t, "Радио-Т 797", twitterNotif.SendCalls()[1].Item.Title)
	assert.Equal(t, "Радио-Т 796", twitterNotif.SendCalls()[2].Item.Title)

	// running processor second time to load new items and remove old ones
	testFeed, err = os.ReadFile("./testdata/rss2.xml")
	require.NoError(t, err)
	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Millisecond*900)
	defer cancel2()

	err = proc.Do(ctx2)
	assert.EqualError(t, err, "context deadline exceeded")

	res, err = boltStore.Load("feed1", 10, false)
	require.NoError(t, err)
	assert.Equal(t, 5, len(res), "5 items in the db: new items loaded, old items removed")
	assert.Equal(t, "Радио-Т 801", res[0].Title)
	assert.Equal(t, "Радио-Т 800", res[1].Title)
	assert.Equal(t, "Радио-Т 799", res[2].Title)
	assert.Equal(t, "Радио-Т 798", res[3].Title)
	assert.Equal(t, "Радио-Т 797", res[4].Title)

	require.Equal(t, 6, len(tgNotif.SendCalls()))
	assert.Equal(t, "Радио-Т 801", tgNotif.SendCalls()[3].Item.Title)
	assert.Equal(t, "tgChannel", tgNotif.SendCalls()[3].ChanID)
	assert.Equal(t, "Радио-Т 800", tgNotif.SendCalls()[4].Item.Title)
	assert.Equal(t, "Радио-Т 799", tgNotif.SendCalls()[5].Item.Title)

	require.Equal(t, 6, len(twitterNotif.SendCalls()))
	assert.Equal(t, "Радио-Т 801", twitterNotif.SendCalls()[3].Item.Title)
	assert.Equal(t, "Радио-Т 800", twitterNotif.SendCalls()[4].Item.Title)
	assert.Equal(t, "Радио-Т 799", twitterNotif.SendCalls()[5].Item.Title)
}

func TestProcessor_DoLoadMaxItems(t *testing.T) {

	tgNotif := &mocks.TelegramNotifMock{SendFunc: func(string, feed.Item) error {
		return nil
	}}

	twitterNotif := &mocks.TwitterNotifMock{SendFunc: func(feed.Item) error {
		return nil
	}}

	tmpfile := filepath.Join(os.TempDir(), "test.db")
	defer os.Remove(tmpfile)

	db, err := bolt.Open(tmpfile, 0o600, &bolt.Options{Timeout: 1 * time.Second})
	require.NoError(t, err)
	boltStore := &BoltDB{DB: db}

	testFeed, err := os.ReadFile("./testdata/rss2.xml")
	require.NoError(t, err)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Length", strconv.Itoa(len(testFeed)))
		_, e := w.Write(testFeed)
		require.NoError(t, e)
	}))
	defer ts.Close()

	proc := Processor{
		Conf: &config.Conf{
			Feeds: map[string]config.Feed{
				"feed1": {
					Title:           "title1",
					Description:     "description",
					Link:            "link",
					Image:           "imageUrl",
					Language:        "language",
					TelegramChannel: "tgChannel",
					Filter:          config.Filter{},
					Sources:         []config.Source{{Name: "sourceName", URL: ts.URL}},
					ExtendDateTitle: "",
					Author:          "author",
				},
			},
			System: struct {
				UpdateInterval      time.Duration `yaml:"update"`
				HTTPResponseTimeout time.Duration `yaml:"http_response_timeout"`
				MaxItems            int           `yaml:"max_per_feed"`
				MaxTotal            int           `yaml:"max_total"`
				MaxKeepInDB         int           `yaml:"max_keep"`
				Concurrent          int           `yaml:"concurrent"`
				BaseURL             string        `yaml:"base_url"`
			}{
				UpdateInterval:      time.Second / 2,
				HTTPResponseTimeout: time.Second,
				MaxItems:            3,
				MaxTotal:            5,
				MaxKeepInDB:         5,
				Concurrent:          1,
				BaseURL:             "baseUrl",
			},
			YouTube: struct {
				DlTemplate      string             `yaml:"dl_template"`
				BaseChanURL     string             `yaml:"base_chan_url"`
				BasePlaylistURL string             `yaml:"base_playlist_url"`
				Channels        []youtube.FeedInfo `yaml:"channels"`
				BaseURL         string             `yaml:"base_url"`
				UpdateInterval  time.Duration      `yaml:"update"`
				MaxItems        int                `yaml:"max_per_channel"`
				FilesLocation   string             `yaml:"files_location"`
				RSSLocation     string             `yaml:"rss_location"`
				SkipShorts      time.Duration      `yaml:"skip_shorts"`
				DisableUpdates  bool               `yaml:"disable_updates"`
				YtDlpUpdate     struct {
					Interval time.Duration `yaml:"interval"`
					Command  string        `yaml:"command"`
				} `yaml:"ytdlp_update"`
			}{},
		},
		Store:         boltStore,
		TelegramNotif: tgNotif,
		TwitterNotif:  twitterNotif,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*900)
	defer cancel()

	// running processor first time to load items to db
	err = proc.Do(ctx)
	assert.EqualError(t, err, "context deadline exceeded")

	res, err := boltStore.Load("feed1", 10, false)
	require.NoError(t, err)
	assert.Equal(t, 3, len(res), "3 items are loaded as per MaxItems")
	assert.Equal(t, "Радио-Т 801", res[0].Title)
	assert.Equal(t, "Радио-Т 800", res[1].Title)
	assert.Equal(t, "Радио-Т 799", res[2].Title)

	require.Equal(t, 3, len(tgNotif.SendCalls()))
	require.Equal(t, 3, len(twitterNotif.SendCalls()))
}

func TestProcessor_DoSkipItems(t *testing.T) {

	tgNotif := &mocks.TelegramNotifMock{SendFunc: func(string, feed.Item) error {
		return nil
	}}

	twitterNotif := &mocks.TwitterNotifMock{SendFunc: func(feed.Item) error {
		return nil
	}}

	tmpfile := filepath.Join(os.TempDir(), "test.db")
	defer os.Remove(tmpfile)

	db, err := bolt.Open(tmpfile, 0o600, &bolt.Options{Timeout: 1 * time.Second})
	require.NoError(t, err)
	boltStore := &BoltDB{DB: db}

	testFeed, err := os.ReadFile("./testdata/rss1.xml")
	require.NoError(t, err)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Length", strconv.Itoa(len(testFeed)))
		_, e := w.Write(testFeed)
		require.NoError(t, e)
	}))
	defer ts.Close()

	proc := Processor{
		Conf: &config.Conf{
			Feeds: map[string]config.Feed{
				"feed1": {
					Title:           "title1",
					Description:     "description",
					Link:            "link",
					Image:           "imageUrl",
					Language:        "language",
					TelegramChannel: "tgChannel",
					Filter: config.Filter{
						Title: "Радио-Т 79[56]",
					},
					Sources:         []config.Source{{Name: "sourceName", URL: ts.URL}},
					ExtendDateTitle: "",
					Author:          "author",
				},
			},
			System: struct {
				UpdateInterval      time.Duration `yaml:"update"`
				HTTPResponseTimeout time.Duration `yaml:"http_response_timeout"`
				MaxItems            int           `yaml:"max_per_feed"`
				MaxTotal            int           `yaml:"max_total"`
				MaxKeepInDB         int           `yaml:"max_keep"`
				Concurrent          int           `yaml:"concurrent"`
				BaseURL             string        `yaml:"base_url"`
			}{
				UpdateInterval:      time.Second / 2,
				HTTPResponseTimeout: time.Second,
				MaxItems:            10,
				MaxTotal:            10,
				MaxKeepInDB:         10,
				Concurrent:          1,
				BaseURL:             "baseUrl",
			},
			YouTube: struct {
				DlTemplate      string             `yaml:"dl_template"`
				BaseChanURL     string             `yaml:"base_chan_url"`
				BasePlaylistURL string             `yaml:"base_playlist_url"`
				Channels        []youtube.FeedInfo `yaml:"channels"`
				BaseURL         string             `yaml:"base_url"`
				UpdateInterval  time.Duration      `yaml:"update"`
				MaxItems        int                `yaml:"max_per_channel"`
				FilesLocation   string             `yaml:"files_location"`
				RSSLocation     string             `yaml:"rss_location"`
				SkipShorts      time.Duration      `yaml:"skip_shorts"`
				DisableUpdates  bool               `yaml:"disable_updates"`
				YtDlpUpdate     struct {
					Interval time.Duration `yaml:"interval"`
					Command  string        `yaml:"command"`
				} `yaml:"ytdlp_update"`
			}{},
		},
		Store:         boltStore,
		TelegramNotif: tgNotif,
		TwitterNotif:  twitterNotif,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*900)
	defer cancel()

	// running processor first time to load items to db
	err = proc.Do(ctx)
	assert.EqualError(t, err, "context deadline exceeded")

	res, err := boltStore.Load("feed1", 10, false)
	require.NoError(t, err)
	assert.Equal(t, 3, len(res), "all 3 items are loaded")
	assert.Equal(t, "Радио-Т 798", res[0].Title)
	assert.Equal(t, "Радио-Т 797", res[1].Title)
	assert.Equal(t, "Радио-Т 796", res[2].Title)

	require.Equal(t, 2, len(tgNotif.SendCalls()))
	assert.Equal(t, "Радио-Т 798", tgNotif.SendCalls()[0].Item.Title)
	assert.Equal(t, "Радио-Т 797", tgNotif.SendCalls()[1].Item.Title)

	require.Equal(t, 2, len(twitterNotif.SendCalls()))
	assert.Equal(t, "Радио-Т 798", twitterNotif.SendCalls()[0].Item.Title)
	assert.Equal(t, "Радио-Т 797", twitterNotif.SendCalls()[1].Item.Title)
}
