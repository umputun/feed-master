package api

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
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

// setupTestServer creates a test server with initialized cache and templates
func setupTestServer(t *testing.T, conf config.Conf, store Store, youtubeStore YoutubeStore) *Server {
	srv := &Server{
		Version:       "1.0",
		Conf:          conf,
		Store:         store,
		YoutubeStore:  youtubeStore,
		TemplLocation: "../webapp/templates/*",
	}

	// initialize cache
	o := lcw.NewOpts[[]byte]()
	cache, err := lcw.NewExpirableCache(o.TTL(time.Minute*3), o.MaxCacheSize(10*1024*1024))
	require.NoError(t, err)
	srv.cache = cache
	srv.loadTemplates()

	return srv
}

func TestServer_getFeedPageCtrl(t *testing.T) {
	conf := config.Conf{
		Feeds: map[string]config.Feed{
			"feed1": {
				Title:           "Test Feed 1",
				Description:     "Test Description",
				Link:            "http://example.com/feed1",
				TelegramChannel: "test_channel",
			},
		},
	}
	conf.System.BaseURL = "http://localhost"

	storeMock := &mocks.StoreMock{}
	items := []feed.Item{
		{
			Title:       "Item 1",
			Link:        "http://example.com/item1",
			Description: "Description 1",
			DT:          time.Date(2025, 8, 3, 12, 0, 0, 0, time.UTC),
			Duration:    "3600",
			Enclosure: feed.Enclosure{
				URL: "http://example.com/audio1.mp3",
			},
		},
		{
			Title:       "Item 2",
			Link:        "http://example.com/item2",
			Description: "Description 2",
			DT:          time.Date(2025, 8, 2, 12, 0, 0, 0, time.UTC),
			Duration:    "1800",
			Junk:        true,
			Enclosure: feed.Enclosure{
				URL: "http://example.com/audio2.mp3",
			},
		},
	}
	storeMock.LoadFunc = func(fmFeed string, maxItems int, skipJunk bool) ([]feed.Item, error) {
		return items, nil
	}

	srv := setupTestServer(t, conf, storeMock, nil)

	ts := httptest.NewServer(srv.router())
	defer ts.Close()

	client := http.Client{Timeout: time.Second}
	resp, err := client.Get(ts.URL + "/feed/feed1")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	body := string(bodyBytes)

	// check key elements
	assert.Contains(t, body, "Test Feed 1")
	assert.Contains(t, body, "Test Description")
	assert.Contains(t, body, "Item 1")
	assert.Contains(t, body, "Item 2")
	assert.Contains(t, body, "http://example.com/audio1.mp3")
	assert.Contains(t, body, "junk-row") // item 2 should have junk class
	assert.Contains(t, body, "t.me/test_channel")

	// check footer with current year
	currentYear := time.Now().Year()
	assert.Contains(t, body, fmt.Sprintf("&copy; %d Umputun", currentYear))
	assert.Contains(t, body, "Open Source, MIT License")
}

func TestServer_getFeedsPageCtrl(t *testing.T) {
	conf := config.Conf{
		Feeds: map[string]config.Feed{
			"feed1": {
				Title:           "Test Feed 1",
				Description:     "Description 1",
				TelegramChannel: "channel1",
				Sources: []config.Source{
					{Name: "Source 1"},
					{Name: "Source 2"},
				},
			},
			"feed2": {
				Title:       "Test Feed 2",
				Description: "Description 2",
				Sources: []config.Source{
					{Name: "Source 3"},
				},
			},
		},
	}
	conf.System.BaseURL = "http://localhost"

	storeMock := &mocks.StoreMock{}
	storeMock.LoadFunc = func(fmFeed string, maxItems int, skipJunk bool) ([]feed.Item, error) {
		return []feed.Item{
			{DT: time.Date(2025, 8, 3, 12, 0, 0, 0, time.UTC)},
		}, nil
	}

	srv := setupTestServer(t, conf, storeMock, nil)

	ts := httptest.NewServer(srv.router())
	defer ts.Close()

	client := http.Client{Timeout: time.Second}
	resp, err := client.Get(ts.URL + "/feeds")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	body := string(bodyBytes)

	// check feeds list
	assert.Contains(t, body, "Test Feed 1")
	assert.Contains(t, body, "Test Feed 2")
	assert.Contains(t, body, "2 feeds")
	assert.Contains(t, body, "2 sources") // feed1 has 2 sources
	assert.Contains(t, body, "1 sources") // feed2 has 1 source
	assert.Contains(t, body, "t.me/channel1")

	// check footer
	currentYear := time.Now().Year()
	assert.Contains(t, body, fmt.Sprintf("&copy; %d Umputun", currentYear))
}

func TestServer_getSourcesPageCtrl(t *testing.T) {
	conf := config.Conf{
		Feeds: map[string]config.Feed{
			"feed1": {
				Sources: []config.Source{
					{Name: "YouTube Channel 1"},
					{Name: "YouTube Channel 2"},
				},
			},
		},
	}

	srv := setupTestServer(t, conf, nil, nil)

	ts := httptest.NewServer(srv.router())
	defer ts.Close()

	client := http.Client{Timeout: time.Second}
	resp, err := client.Get(ts.URL + "/feed/feed1/sources")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	body := string(bodyBytes)

	// check sources
	assert.Contains(t, body, "YouTube Channel 1")
	assert.Contains(t, body, "YouTube Channel 2")
	assert.Contains(t, body, "2 sources")

	// check footer
	currentYear := time.Now().Year()
	assert.Contains(t, body, fmt.Sprintf("&copy; %d Umputun", currentYear))
}

func TestServer_getFeedSourceCtrl(t *testing.T) {
	conf := config.Conf{
		Feeds: map[string]config.Feed{
			"feed1": {},
		},
	}
	conf.YouTube.Channels = []youtube.FeedInfo{
		{
			ID:   "channel1",
			Name: "Test Channel",
			Type: ytfeed.FTChannel,
		},
	}
	conf.YouTube.BaseURL = "http://localhost/yt"
	conf.System.BaseURL = "http://localhost"

	ytStoreMock := &mocks.YoutubeStoreMock{}
	entries := []ytfeed.Entry{
		{
			Title:     "Video 1",
			VideoID:   "vid1",
			ChannelID: "channel1",
			Published: time.Date(2025, 8, 3, 12, 0, 0, 0, time.UTC),
			Duration:  3600,
			File:      "/path/to/file1.mp3",
			Link: struct {
				Href string `xml:"href,attr"`
			}{Href: "https://youtube.com/watch?v=vid1"},
			Media: struct {
				Description template.HTML `xml:"description"`
				Thumbnail   struct {
					URL string `xml:"url,attr"`
				} `xml:"thumbnail"`
			}{Description: "Video 1 description"},
		},
	}
	ytStoreMock.LoadFunc = func(channelID string, maxItems int) ([]ytfeed.Entry, error) {
		return entries, nil
	}

	srv := setupTestServer(t, conf, nil, ytStoreMock)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /feed/{name}/source/{source}", srv.getFeedSourceCtrl)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	client := http.Client{Timeout: time.Second}
	resp, err := client.Get(ts.URL + "/feed/feed1/source/Test%20Channel")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	body := string(bodyBytes)

	// check content
	assert.Contains(t, body, "Test Channel")
	assert.Contains(t, body, "Video 1")
	assert.Contains(t, body, "http://localhost/yt/file1.mp3")
	assert.Contains(t, body, "1h0m0s") // duration formatted

	// check footer
	currentYear := time.Now().Year()
	assert.Contains(t, body, fmt.Sprintf("&copy; %d Umputun", currentYear))
}

func TestServer_getYoutubeChannelsPageCtrl(t *testing.T) {
	conf := config.Conf{}
	conf.YouTube.Channels = []youtube.FeedInfo{
		{
			ID:   "channel1",
			Name: "Channel 1",
			Type: ytfeed.FTChannel,
		},
		{
			ID:   "playlist1",
			Name: "Playlist 1",
			Type: ytfeed.FTPlaylist,
		},
	}
	conf.YouTube.BaseChanURL = "https://www.youtube.com/feeds/videos.xml?channel_id="
	conf.YouTube.BasePlaylistURL = "https://www.youtube.com/feeds/videos.xml?playlist_id="

	ytStoreMock := &mocks.YoutubeStoreMock{}
	ytStoreMock.LoadFunc = func(channelID string, maxItems int) ([]ytfeed.Entry, error) {
		return []ytfeed.Entry{
			{Published: time.Date(2025, 8, 3, 12, 0, 0, 0, time.UTC)},
		}, nil
	}

	srv := setupTestServer(t, conf, nil, ytStoreMock)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /yt/channels", srv.getYoutubeChannelsPageCtrl)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	client := http.Client{Timeout: time.Second}
	resp, err := client.Get(ts.URL + "/yt/channels")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	body := string(bodyBytes)

	// check channels
	assert.Contains(t, body, "Channel 1")
	assert.Contains(t, body, "Playlist 1")
	assert.Contains(t, body, "2 channels")
	assert.Contains(t, body, "https://youtube.com/channel/channel1")
	assert.Contains(t, body, "https://www.youtube.com/playlist?list=playlist1")

	// check footer
	currentYear := time.Now().Year()
	assert.Contains(t, body, fmt.Sprintf("&copy; %d Umputun", currentYear))
}

func TestServer_renderErrorPage(t *testing.T) {
	srv := setupTestServer(t, config.Conf{}, nil, nil)

	tests := []struct {
		name       string
		err        error
		statusCode int
		wantBody   []string
		notWant    []string
	}{
		{
			name:       "404 error",
			err:        fmt.Errorf("test error message"),
			statusCode: 404,
			wantBody:   []string{"404", "test error message", "Something went wrong!"},
			notWant:    []string{"©"},
		},
		{
			name:       "500 error",
			err:        fmt.Errorf("internal server error"),
			statusCode: 500,
			wantBody:   []string{"500", "internal server error", "Something went wrong!"},
			notWant:    []string{"©"},
		},
		{
			name:       "403 with detailed message",
			err:        fmt.Errorf("forbidden: insufficient permissions"),
			statusCode: 403,
			wantBody:   []string{"403", "forbidden: insufficient permissions", "Something went wrong!"},
			notWant:    []string{"©"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/test", http.NoBody)

			srv.renderErrorPage(rec, req, tt.err, tt.statusCode)

			// renderErrorPage doesn't set status code, it just renders template
			assert.Equal(t, http.StatusOK, rec.Code)
			body := rec.Body.String()

			for _, want := range tt.wantBody {
				assert.Contains(t, body, want)
			}

			for _, notWant := range tt.notWant {
				assert.NotContains(t, body, notWant)
			}
		})
	}
}

func TestTemplateCurrentYear(t *testing.T) {
	srv := &Server{TemplLocation: "../webapp/templates/*"}
	srv.loadTemplates()

	// test that currentYear function returns current year
	tmpl := srv.templates.Lookup("footer")
	require.NotNil(t, tmpl)

	var buf bytes.Buffer
	err := tmpl.Execute(&buf, nil)
	require.NoError(t, err)

	currentYear := time.Now().Year()
	assert.Contains(t, buf.String(), fmt.Sprintf("&copy; %d", currentYear))
	assert.Contains(t, buf.String(), "Umputun")
	assert.Contains(t, buf.String(), "Open Source, MIT License")
}
