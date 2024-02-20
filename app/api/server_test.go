package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-pkgz/lcw/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/feed-master/app/api/mocks"
	"github.com/umputun/feed-master/app/config"
	"github.com/umputun/feed-master/app/feed"
	"github.com/umputun/feed-master/app/youtube"
	ytfeed "github.com/umputun/feed-master/app/youtube/feed"
)

func TestServer_Run(t *testing.T) {
	s := Server{Version: "1.0", TemplLocation: "../webapp/templates/*"}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	port := rand.Intn(10000) + 4000 // nolint
	go func() {
		time.Sleep(time.Millisecond * 100)
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/ping", port))
		require.NoError(t, err)
		defer resp.Body.Close() // nolint
		require.Equal(t, http.StatusOK, resp.StatusCode)
		t.Logf("%+v", resp.Header)
		assert.Equal(t, "1.0", resp.Header.Get("App-Version"))
		assert.Equal(t, "feed-master", resp.Header.Get("App-Name"))
	}()
	s.Run(ctx, port)
}

func TestServer_getFeedCtrl(t *testing.T) {

	store := &mocks.StoreMock{
		LoadFunc: func(string, int, bool) ([]feed.Item, error) {
			return []feed.Item{
				{
					GUID:        "guid1",
					Title:       "title1",
					Link:        "http://example.com/link1",
					Description: "some description1",
					Enclosure: feed.Enclosure{
						URL:    "http://example.com/enclosure1",
						Type:   "audio/mpeg",
						Length: 12345,
					},
				},
				{
					GUID:        "guid2",
					Title:       "title2",
					Link:        "http://example.com/link2",
					Description: "some description2",
					Enclosure: feed.Enclosure{
						URL:    "http://example.com/enclosure2",
						Type:   "audio/mpeg",
						Length: 12346,
					},
				},
			}, nil
		},
	}

	s := Server{
		Version:       "1.0",
		TemplLocation: "../webapp/templates/*",
		Store:         store,
		cache:         lcw.NewNopCache[[]byte](),
		Conf: config.Conf{
			Feeds: map[string]config.Feed{
				"feed1": {
					Title:       "feed1",
					Language:    "ru-ru",
					Description: "this is feed1",
					Link:        "http://example.com/feed1",
					Author:      "Feed Master",
					OwnerEmail:  "test@email.com",
				},
				"feed2": {
					Title: "feed2",
				},
			},
		},
	}
	ts := httptest.NewServer(s.router())
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL + "/rss/feed1")
	require.NoError(t, err)
	defer resp.Body.Close() // nolint
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	body := string(respBody)
	t.Logf("resp body: %s", body)
	assert.Contains(t, body, "<title>feed1</title>")
	assert.Contains(t, body, "<language>ru-ru</language>")
	assert.Contains(t, body, " <description>this is feed1</description>")
	assert.Contains(t, body, "<itunes:author>Feed Master</itunes:author>")
	assert.Contains(t, body, "<itunes:explicit>no</itunes:explicit>")
	assert.Contains(t, body, "<itunes:email>test@email.com</itunes:email>")
	assert.Contains(t, body, "<itunes:name>Feed Master</itunes:name>")
	assert.Contains(t, body, "<guid>guid1</guid>")
	assert.Contains(t, body, "<title>title1</title>")
	assert.Contains(t, body, "<link>http://example.com/link1</link>")
	assert.Contains(t, body, "<description>some description1</description>")
	assert.Contains(t, body, `<enclosure url="http://example.com/enclosure1" length="12345" type="audio/mpeg"></enclosure>`)
	assert.NotContains(t, body, `<itunes:image href=""></itunes:image>`)
	assert.NotContains(t, body, `<media:thumbnail url=""></media:thumbnail>`)

	assert.Equal(t, 1, len(store.LoadCalls()))
	assert.Equal(t, "feed1", store.LoadCalls()[0].FmFeed)
}

func TestServer_getFeedCtrlExtendDateTitle(t *testing.T) {

	store := &mocks.StoreMock{
		LoadFunc: func(string, int, bool) ([]feed.Item, error) {
			return []feed.Item{
				{
					GUID:        "guid1",
					Title:       "title1",
					Link:        "http://example.com/link1",
					Description: "some description1",
					DT:          time.Date(2022, time.April, 3, 16, 30, 0, 0, time.UTC),
					Enclosure: feed.Enclosure{
						URL:    "http://example.com/enclosure1",
						Type:   "audio/mpeg",
						Length: 12345,
					},
				},
				{
					GUID:        "guid2",
					Title:       "title2",
					Link:        "http://example.com/link2",
					Description: "some description2",
					Enclosure: feed.Enclosure{
						URL:    "http://example.com/enclosure2",
						Type:   "audio/mpeg",
						Length: 12346,
					},
				},
			}, nil
		},
	}

	s := Server{
		Version:       "1.0",
		TemplLocation: "../webapp/templates/*",
		Store:         store,
		cache:         lcw.NewNopCache[[]byte](),
		Conf: config.Conf{
			Feeds: map[string]config.Feed{
				"feed1": {
					Title:           "feed1",
					Language:        "ru-ru",
					Description:     "this is feed1",
					Link:            "http://example.com/feed1",
					ExtendDateTitle: "yyyymmdd",
				},
			},
		},
	}
	ts := httptest.NewServer(s.router())
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL + "/rss/feed1")
	require.NoError(t, err)
	defer resp.Body.Close() // nolint
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	body := string(respBody)
	t.Logf("resp body: %s", body)
	assert.Contains(t, body, "<title>feed1</title>")
	assert.Contains(t, body, "<language>ru-ru</language>")
	assert.Contains(t, body, " <description>this is feed1</description>")

	assert.Contains(t, body, "<guid>guid1</guid>")
	assert.Contains(t, body, "<title>title1 (2022-04-03)</title>")

	assert.Equal(t, 1, len(store.LoadCalls()))
	assert.Equal(t, "feed1", store.LoadCalls()[0].FmFeed)
}

func TestServer_getFeedCtrlFeedImage(t *testing.T) {

	store := &mocks.StoreMock{
		LoadFunc: func(string, int, bool) ([]feed.Item, error) {
			return []feed.Item{
				{
					GUID:        "guid1",
					Title:       "title1",
					Link:        "http://example.com/link1",
					Description: "some description1",
					Enclosure: feed.Enclosure{
						URL:    "http://example.com/enclosure1",
						Type:   "audio/mpeg",
						Length: 12345,
					},
				},
				{
					GUID:        "guid2",
					Title:       "title2",
					Link:        "http://example.com/link2",
					Description: "some description2",
					Enclosure: feed.Enclosure{
						URL:    "http://example.com/enclosure2",
						Type:   "audio/mpeg",
						Length: 12346,
					},
				},
			}, nil
		},
	}

	s := Server{
		Version:       "1.0",
		TemplLocation: "../webapp/templates/*",
		Store:         store,
		cache:         lcw.NewNopCache[[]byte](),
		Conf: config.Conf{
			Feeds: map[string]config.Feed{
				"feed1": {
					Title:       "feed1",
					Language:    "ru-ru",
					Description: "this is feed1",
					Link:        "http://example.com/feed1",
				},
				"feed2": {
					Title: "feed2",
				},
			},
		},
	}
	s.Conf.System.BaseURL = "http://example.com"

	ts := httptest.NewServer(s.router())
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL + "/rss/feed1")
	require.NoError(t, err)
	defer resp.Body.Close() // nolint
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	body := string(respBody)
	t.Logf("resp body: %s", body)
	assert.Contains(t, body, "<title>feed1</title>")

	assert.Contains(t, body, "<guid>guid1</guid>")
	assert.Contains(t, body, `<itunes:image href="http://example.com/images/feed1"></itunes:image>`)
	assert.Contains(t, body, `<media:thumbnail url="http://example.com/images/feed1"></media:thumbnail>`)

	assert.Equal(t, 1, len(store.LoadCalls()))
	assert.Equal(t, "feed1", store.LoadCalls()[0].FmFeed)
}

func TestServer_regenerateRSSCtrl(t *testing.T) {

	yt := &mocks.YoutubeSvcMock{
		RSSFeedFunc: func(youtube.FeedInfo) (string, error) {
			return "blah", nil
		},
		StoreRSSFunc: func(string, string) error {
			return nil
		},
	}

	s := Server{
		Version:       "1.0",
		TemplLocation: "../webapp/templates/*",
		YoutubeSvc:    yt,
		Conf:          config.Conf{},
		AdminPasswd:   "123456",
	}
	s.Conf.YouTube.Channels = []youtube.FeedInfo{{ID: "chan1"}, {ID: "chan2"}}
	ts := httptest.NewServer(s.router())
	defer ts.Close()

	{
		req, err := http.NewRequest("POST", ts.URL+"/yt/rss/generate", bytes.NewBuffer(nil))
		require.NoError(t, err)
		req.SetBasicAuth("admin", "bad")
		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close() // nolint
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	}

	{
		req, err := http.NewRequest("POST", ts.URL+"/yt/rss/generate", bytes.NewBuffer(nil))
		require.NoError(t, err)
		req.SetBasicAuth("admin", "123456")
		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close() // nolint
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}

	require.Equal(t, 2, len(yt.RSSFeedCalls()))
	assert.Equal(t, "chan1", yt.RSSFeedCalls()[0].Cinfo.ID)
	assert.Equal(t, "chan2", yt.RSSFeedCalls()[1].Cinfo.ID)

	require.Equal(t, 2, len(yt.StoreRSSCalls()))
	require.Equal(t, "chan1", yt.StoreRSSCalls()[0].ChanID)
	require.Equal(t, "blah", yt.StoreRSSCalls()[0].Rss)
	require.Equal(t, "chan2", yt.StoreRSSCalls()[1].ChanID)
	require.Equal(t, "blah", yt.StoreRSSCalls()[1].Rss)
}

func TestServer_removeEntryCtrl(t *testing.T) {
	yt := &mocks.YoutubeSvcMock{
		RemoveEntryFunc: func(ytfeed.Entry) error {
			return nil
		},
	}

	s := Server{
		Version:       "1.0",
		TemplLocation: "../webapp/templates/*",
		YoutubeSvc:    yt,
		Conf:          config.Conf{},
		AdminPasswd:   "123456",
	}

	ts := httptest.NewServer(s.router())
	defer ts.Close()

	{
		req, err := http.NewRequest("DELETE", ts.URL+"/yt/entry/chan1/vid1", bytes.NewBuffer(nil))
		require.NoError(t, err)
		req.SetBasicAuth("admin", "bad")
		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close() // nolint
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	}

	{
		req, err := http.NewRequest("DELETE", ts.URL+"/yt/entry/chan1/vid1", bytes.NewBuffer(nil))
		require.NoError(t, err)
		req.SetBasicAuth("admin", "123456")
		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close() // nolint
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}

	require.Equal(t, 1, len(yt.RemoveEntryCalls()))
	require.Equal(t, "chan1", yt.RemoveEntryCalls()[0].Entry.ChannelID)
	require.Equal(t, "vid1", yt.RemoveEntryCalls()[0].Entry.VideoID)
}

func TestServer_configCtrl(t *testing.T) {

	store := &mocks.StoreMock{}

	s := Server{
		Version:       "1.0",
		TemplLocation: "../webapp/templates/*",
		Store:         store,
		cache:         lcw.NewNopCache[[]byte](),
		Conf: config.Conf{
			Feeds: map[string]config.Feed{
				"feed1": {
					Title:       "feed1",
					Language:    "ru-ru",
					Description: "this is feed1",
					Link:        "http://example.com/feed1",
					Author:      "Feed Master",
					OwnerEmail:  "test@email.com",
				},
				"feed2": {
					Title: "feed2",
				},
			},
		},
	}
	ts := httptest.NewServer(s.router())
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL + "/config")
	require.NoError(t, err)
	defer resp.Body.Close() // nolint
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	body := string(respBody)
	t.Logf("resp body: %s", body)
	assert.Contains(t, body, "feed1")
	assert.Contains(t, body, "feed2")
	assert.Contains(t, body, "this is feed1")
	assert.Contains(t, body, "http://example.com/feed1")
}
