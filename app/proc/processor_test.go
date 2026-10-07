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

func TestProcessor_ProcessAtomFeed(t *testing.T) {
	published := time.Now().UTC().Truncate(time.Second)
	timestamp := published.Format(time.RFC3339)
	content := `<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Example feed</title>
  <id>urn:example:feed</id>
  <updated>` + timestamp + `</updated>
  <entry>
    <title>First entry</title>
    <id>urn:example:first</id>
    <link href="https://example.org/first"/>
    <updated>` + timestamp + `</updated>
    <summary>First content</summary>
  </entry>
  <entry>
    <title>Second entry</title>
    <id>urn:example:second</id>
    <link href="https://example.org/second"/>
    <updated>` + timestamp + `</updated>
    <summary>Second content</summary>
  </entry>
</feed>`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml")
		_, err := w.Write([]byte(content))
		assert.NoError(t, err)
	}))
	defer ts.Close()

	db, err := bolt.Open(filepath.Join(t.TempDir(), "atom.db"), 0o600, &bolt.Options{Timeout: time.Second})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	tgNotif := &mocks.TelegramNotifMock{SendFunc: func(string, feed.Item) error { return nil }}
	twitterNotif := &mocks.TwitterNotifMock{SendFunc: func(feed.Item) error { return nil }}
	conf := &config.Conf{}
	conf.System.MaxKeepInDB = 10
	processor := Processor{
		Conf:          conf,
		Store:         &BoltDB{DB: db},
		TelegramNotif: tgNotif,
		TwitterNotif:  twitterNotif,
	}

	processor.processFeed("atom", ts.URL, "channel", 10, config.Filter{})
	items, err := processor.Store.Load("atom", 10, false)
	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.ElementsMatch(t, []string{"urn:example:first", "urn:example:second"}, []string{items[0].GUID, items[1].GUID})
	for _, item := range items {
		assert.Equal(t, published.Format(time.RFC1123Z), item.PubDate)
		assert.True(t, published.Equal(item.DT), "got stored time %s", item.DT)
	}
	require.Len(t, tgNotif.SendCalls(), 2)
	require.Len(t, twitterNotif.SendCalls(), 2)

	processor.processFeed("atom", ts.URL, "channel", 10, config.Filter{})
	items, err = processor.Store.Load("atom", 10, false)
	require.NoError(t, err)
	assert.Len(t, items, 2)
	assert.Len(t, tgNotif.SendCalls(), 2)
	assert.Len(t, twitterNotif.SendCalls(), 2)
}

func TestProcessor_DoRemoveOldItems(t *testing.T) {
	lgr.Setup(lgr.Debug)
	tgNotif := &mocks.TelegramNotifMock{SendFunc: func(string, feed.Item) error {
		return nil
	}}

	twitterNotif := &mocks.TwitterNotifMock{SendFunc: func(feed.Item) error {
		return nil
	}}

	tmpfile := filepath.Join(t.TempDir(), "test.db")

	db, err := bolt.Open(tmpfile, 0o600, &bolt.Options{Timeout: 2 * time.Second})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	boltStore := &BoltDB{DB: db}

	testFeed, err := os.ReadFile("./testdata/rss1.xml")
	require.NoError(t, err)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Length", strconv.Itoa(len(testFeed)))
		_, e := w.Write(testFeed)
		assert.NoError(t, e)
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
					Interval       time.Duration `yaml:"interval"`
					Command        string        `yaml:"command"`
					ForceOnStartup bool          `yaml:"force_on_startup"`
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
	require.EqualError(t, err, "processor stopped: context deadline exceeded")

	res, err := boltStore.Load("feed1", 10, false)
	require.NoError(t, err)
	assert.Len(t, res, 3, "all 3 items loaded on the first rss parsing")
	assert.Equal(t, "Радио-Т 798", res[0].Title)
	assert.Equal(t, "Радио-Т 797", res[1].Title)
	assert.Equal(t, "Радио-Т 796", res[2].Title)

	require.Len(t, tgNotif.SendCalls(), 3)
	assert.Equal(t, "Радио-Т 798", tgNotif.SendCalls()[0].Item.Title)
	assert.Equal(t, "tgChannel", tgNotif.SendCalls()[0].ChanID)
	assert.Equal(t, "Радио-Т 797", tgNotif.SendCalls()[1].Item.Title)
	assert.Equal(t, "Радио-Т 796", tgNotif.SendCalls()[2].Item.Title)

	require.Len(t, twitterNotif.SendCalls(), 3)
	assert.Equal(t, "Радио-Т 798", twitterNotif.SendCalls()[0].Item.Title)
	assert.Equal(t, "Радио-Т 797", twitterNotif.SendCalls()[1].Item.Title)
	assert.Equal(t, "Радио-Т 796", twitterNotif.SendCalls()[2].Item.Title)

	// running processor second time to load new items and remove old ones
	testFeed, err = os.ReadFile("./testdata/rss2.xml")
	require.NoError(t, err)
	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Millisecond*900)
	defer cancel2()

	err = proc.Do(ctx2)
	require.EqualError(t, err, "processor stopped: context deadline exceeded")

	res, err = boltStore.Load("feed1", 10, false)
	require.NoError(t, err)
	assert.Len(t, res, 5, "5 items in the db: new items loaded, old items removed")
	assert.Equal(t, "Радио-Т 801", res[0].Title)
	assert.Equal(t, "Радио-Т 800", res[1].Title)
	assert.Equal(t, "Радио-Т 799", res[2].Title)
	assert.Equal(t, "Радио-Т 798", res[3].Title)
	assert.Equal(t, "Радио-Т 797", res[4].Title)

	require.Len(t, tgNotif.SendCalls(), 6)
	assert.Equal(t, "Радио-Т 801", tgNotif.SendCalls()[3].Item.Title)
	assert.Equal(t, "tgChannel", tgNotif.SendCalls()[3].ChanID)
	assert.Equal(t, "Радио-Т 800", tgNotif.SendCalls()[4].Item.Title)
	assert.Equal(t, "Радио-Т 799", tgNotif.SendCalls()[5].Item.Title)

	require.Len(t, twitterNotif.SendCalls(), 6)
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

	tmpfile := filepath.Join(t.TempDir(), "test.db")

	db, err := bolt.Open(tmpfile, 0o600, &bolt.Options{Timeout: 1 * time.Second})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	boltStore := &BoltDB{DB: db}

	testFeed, err := os.ReadFile("./testdata/rss2.xml")
	require.NoError(t, err)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Length", strconv.Itoa(len(testFeed)))
		_, e := w.Write(testFeed)
		assert.NoError(t, e)
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
					Interval       time.Duration `yaml:"interval"`
					Command        string        `yaml:"command"`
					ForceOnStartup bool          `yaml:"force_on_startup"`
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
	require.EqualError(t, err, "processor stopped: context deadline exceeded")

	res, err := boltStore.Load("feed1", 10, false)
	require.NoError(t, err)
	assert.Len(t, res, 3, "3 items are loaded as per MaxItems")
	assert.Equal(t, "Радио-Т 801", res[0].Title)
	assert.Equal(t, "Радио-Т 800", res[1].Title)
	assert.Equal(t, "Радио-Т 799", res[2].Title)

	require.Len(t, tgNotif.SendCalls(), 3)
	require.Len(t, twitterNotif.SendCalls(), 3)
}

func TestProcessor_DoSkipItems(t *testing.T) {
	tgNotif := &mocks.TelegramNotifMock{SendFunc: func(string, feed.Item) error {
		return nil
	}}

	twitterNotif := &mocks.TwitterNotifMock{SendFunc: func(feed.Item) error {
		return nil
	}}

	tmpfile := filepath.Join(t.TempDir(), "test.db")

	db, err := bolt.Open(tmpfile, 0o600, &bolt.Options{Timeout: 1 * time.Second})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	boltStore := &BoltDB{DB: db}

	testFeed, err := os.ReadFile("./testdata/rss1.xml")
	require.NoError(t, err)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Length", strconv.Itoa(len(testFeed)))
		_, e := w.Write(testFeed)
		assert.NoError(t, e)
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
					Interval       time.Duration `yaml:"interval"`
					Command        string        `yaml:"command"`
					ForceOnStartup bool          `yaml:"force_on_startup"`
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
	require.EqualError(t, err, "processor stopped: context deadline exceeded")

	res, err := boltStore.Load("feed1", 10, false)
	require.NoError(t, err)
	assert.Len(t, res, 3, "all 3 items are loaded")
	assert.Equal(t, "Радио-Т 798", res[0].Title)
	assert.Equal(t, "Радио-Т 797", res[1].Title)
	assert.Equal(t, "Радио-Т 796", res[2].Title)

	require.Len(t, tgNotif.SendCalls(), 2)
	assert.Equal(t, "Радио-Т 798", tgNotif.SendCalls()[0].Item.Title)
	assert.Equal(t, "Радио-Т 797", tgNotif.SendCalls()[1].Item.Title)

	require.Len(t, twitterNotif.SendCalls(), 2)
	assert.Equal(t, "Радио-Т 798", twitterNotif.SendCalls()[0].Item.Title)
	assert.Equal(t, "Радио-Т 797", twitterNotif.SendCalls()[1].Item.Title)
}
