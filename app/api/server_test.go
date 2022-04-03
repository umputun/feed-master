package api

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/feed-master/app/api/mocks"
	"github.com/umputun/feed-master/app/config"
	"github.com/umputun/feed-master/app/feed"
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
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		t.Logf("%+v", resp.Header)
		assert.Equal(t, "1.0", resp.Header.Get("App-Version"))
		assert.Equal(t, "feed-master", resp.Header.Get("App-Name"))
	}()
	s.Run(ctx, port)
}

func TestServer_getFeedCtrl(t *testing.T) {

	store := &mocks.StoreMock{
		LoadFunc: func(fmFeed string, max int, skipJunk bool) ([]feed.Item, error) {
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
	ts := httptest.NewServer(s.router())
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL + "/rss/feed1")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	t.Logf("resp body: %s", string(body))
	assert.Contains(t, string(body), "<title>feed1</title>")
	assert.Contains(t, string(body), "<language>ru-ru</language>")
	assert.Contains(t, string(body), " <description>this is feed1</description>")

	assert.Contains(t, string(body), "<guid>guid1</guid>")
	assert.Contains(t, string(body), "<title>title1</title>")
	assert.Contains(t, string(body), "<link>http://example.com/link1</link>")
	assert.Contains(t, string(body), "<description>some description1</description>")
	assert.Contains(t, string(body), `<enclosure url="http://example.com/enclosure1" length="12345" type="audio/mpeg"></enclosure>`)

	assert.Equal(t, 1, len(store.LoadCalls()))
	assert.Equal(t, "feed1", store.LoadCalls()[0].FmFeed)
}

func TestServer_getFeedCtrlExtendDateTitle(t *testing.T) {

	store := &mocks.StoreMock{
		LoadFunc: func(fmFeed string, max int, skipJunk bool) ([]feed.Item, error) {
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
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	t.Logf("resp body: %s", string(body))
	assert.Contains(t, string(body), "<title>feed1</title>")
	assert.Contains(t, string(body), "<language>ru-ru</language>")
	assert.Contains(t, string(body), " <description>this is feed1</description>")

	assert.Contains(t, string(body), "<guid>guid1</guid>")
	assert.Contains(t, string(body), "<title>title1 (2022-04-03)</title>")

	assert.Equal(t, 1, len(store.LoadCalls()))
	assert.Equal(t, "feed1", store.LoadCalls()[0].FmFeed)
}
