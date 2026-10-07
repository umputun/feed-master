package feed

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannel_Get(t *testing.T) {
	feedXML, err := os.ReadFile("testdata/channel.xml")
	require.NoError(t, err)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("req: %v", r.URL.String())
		assert.Equal(t, "/blah?channel_id=UCPU28A9z_ka_R5dQfecHJlA", r.URL.String())
		_, e := w.Write(feedXML)
		assert.NoError(t, e)
	}))

	c := Feed{Client: &http.Client{},
		ChannelBaseURL: ts.URL + "/blah?channel_id=", PlaylistBaseURL: ts.URL + "/blah?playlist_id="}

	res, err := c.Get(context.Background(), "UCPU28A9z_ka_R5dQfecHJlA", FTChannel)
	require.NoError(t, err)
	assert.Len(t, res, 15)

	first := res[0]
	assert.Equal(t, "UCPU28A9z_ka_R5dQfecHJlA", first.ChannelID)
	assert.Equal(t, "Hou7PjJR498", first.VideoID)
	assert.Equal(t, "2022-03-20T12:00:07Z", first.Published.Format(time.RFC3339))
	assert.Equal(t, "2022-03-20T12:00:07Z", first.Updated.Format(time.RFC3339))
	assert.Equal(t, "https://www.youtube.com/watch?v=Hou7PjJR498", first.Link.Href)
	assert.Equal(t, "https://www.youtube.com/channel/UCPU28A9z_ka_R5dQfecHJlA", first.Author.URI)
	assert.Equal(t, "RTVI Новости", first.Author.Name)
	assert.Equal(t, `«Мировая война была неизбежна» / Что это было, Максим Шевченко`, first.Title)
	assert.Equal(t, "https://i1.ytimg.com/vi/Hou7PjJR498/hqdefault.jpg", first.Media.Thumbnail.URL)
	assert.Contains(t, first.Media.Description, " Что это было")

	last := res[14]
	assert.Equal(t, "UCPU28A9z_ka_R5dQfecHJlA", last.ChannelID)
	assert.Equal(t, "zBwM0SU1vRk", last.VideoID)
	assert.Equal(t, "2022-03-16T08:00:29Z", last.Published.Format(time.RFC3339))
	assert.Equal(t, "2022-03-16T17:31:46Z", last.Updated.Format(time.RFC3339))
	assert.Equal(t, "https://www.youtube.com/watch?v=zBwM0SU1vRk", last.Link.Href)
	assert.Equal(t, "https://www.youtube.com/channel/UCPU28A9z_ka_R5dQfecHJlA", last.Author.URI)
	assert.Equal(t, "RTVI Новости", last.Author.Name)
	assert.Equal(t, `«Она показала пример». Константин Калачев — об антивоенной акции Овсянниковой в эфире Первого канала`, last.Title)
	assert.Equal(t, "https://i3.ytimg.com/vi/zBwM0SU1vRk/hqdefault.jpg", last.Media.Thumbnail.URL)
	assert.Contains(t, last.Media.Description, "за призыв к публичным несанкционированным акциям протеста")
}

func TestFeed_url(t *testing.T) {
	tbl := []struct {
		ID       string
		feedType Type
		res      string
		err      error
	}{
		{"xyz", FTChannel, "https://www.youtube.com/feeds/videos.xml?channel_id=xyz", nil},
		{"123", FTPlaylist, "https://www.youtube.com/feeds/videos.xml?playlist_id=123", nil},
		{"blah", FTDefault, "https://www.youtube.com/feeds/videos.xml?channel_id=blah", nil},
		{"foo", Type("xxxx"), "", errors.New("unknown feed type xxxx")},
	}

	c := Feed{
		ChannelBaseURL:  "https://www.youtube.com/feeds/videos.xml?channel_id=",
		PlaylistBaseURL: "https://www.youtube.com/feeds/videos.xml?playlist_id=",
	}
	for _, tt := range tbl {
		t.Run(tt.ID, func(t *testing.T) {
			res, err := c.url(tt.ID, tt.feedType)
			if err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.Equal(t, tt.res, res)
			}
		})
	}
}

func TestFeed_GetFallsBackToAPI(t *testing.T) {
	apiJSON, err := os.ReadFile("testdata/playlist_items.json")
	require.NoError(t, err)

	feedSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer feedSrv.Close()

	var gotPlaylistID, gotKey string
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/playlistItems", r.URL.Path)
		assert.Empty(t, r.URL.Query().Get("key"), "api key must not be in the url")
		gotPlaylistID = r.URL.Query().Get("playlistId")
		gotKey = r.Header.Get("X-Goog-Api-Key")
		if gotKey != "good-key" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_, e := w.Write(apiJSON)
		assert.NoError(t, e)
	}))
	defer apiSrv.Close()

	tbl := []struct {
		name           string
		id             string
		feedType       Type
		apiKey         string
		wantPlaylistID string
		wantErr        bool
	}{
		{name: "channel", id: "UCPU28A9z_ka_R5dQfecHJlA", feedType: FTChannel, apiKey: "good-key",
			wantPlaylistID: "UUPU28A9z_ka_R5dQfecHJlA"},
		{name: "default type is a channel", id: "UCPU28A9z_ka_R5dQfecHJlA", feedType: FTDefault, apiKey: "good-key",
			wantPlaylistID: "UUPU28A9z_ka_R5dQfecHJlA"},
		{name: "playlist", id: "PLZHQObOWTQDPD3MizzM2xVFitgF8hE_ab", feedType: FTPlaylist, apiKey: "good-key",
			wantPlaylistID: "PLZHQObOWTQDPD3MizzM2xVFitgF8hE_ab"},
		{name: "no api key keeps the feed error", id: "UCPU28A9z_ka_R5dQfecHJlA", feedType: FTChannel, wantErr: true},
		{name: "api error", id: "UCPU28A9z_ka_R5dQfecHJlA", feedType: FTChannel, apiKey: "bad-key", wantErr: true},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			gotPlaylistID, gotKey = "", ""
			c := Feed{Client: &http.Client{}, ChannelBaseURL: feedSrv.URL + "/?channel_id=",
				PlaylistBaseURL: feedSrv.URL + "/?playlist_id=", APIKey: tt.apiKey, APIBaseURL: apiSrv.URL}

			res, err := c.Get(context.Background(), tt.id, tt.feedType)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantPlaylistID, gotPlaylistID)
			assert.Equal(t, tt.apiKey, gotKey)

			require.Len(t, res, 2, "private video without a publish date is skipped")
			first := res[0]
			assert.Equal(t, "vid3", first.VideoID, "newest first")
			assert.Equal(t, tt.id, first.ChannelID)
			assert.Equal(t, "Newer video", first.Title)
			assert.Equal(t, "2026-09-25T13:35:02Z", first.Published.Format(time.RFC3339))
			assert.Equal(t, first.Published, first.Updated)
			assert.Equal(t, "https://www.youtube.com/watch?v=vid3", first.Link.Href)
			assert.Equal(t, "https://i.ytimg.com/vi/vid3/hqdefault.jpg", first.Media.Thumbnail.URL)
			assert.Equal(t, "Test Channel", first.Author.Name)
			assert.Equal(t, "https://www.youtube.com/channel/UCPU28A9z_ka_R5dQfecHJlA", first.Author.URI)
			assert.Equal(t, "vid1", res[1].VideoID)
			assert.Equal(t, "first line &amp; second", string(res[1].Media.Description))
		})
	}
}
