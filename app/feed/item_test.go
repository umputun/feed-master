package feed

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetFilename(t *testing.T) {
	tbl := []struct {
		url, expected string
	}{
		{"https://example.com/100500/song.mp3", "song.mp3"},
		{"https://example.com//song.mp3", "song.mp3"},
		{"https://example.com/song.mp3", "song.mp3"},
		{"https://example.com/song.mp3/", ""},
		{"https://example.com/", ""},
	}

	for i, tt := range tbl {
		i := i
		tt := tt
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			item := Item{Enclosure: Enclosure{URL: tt.url}}
			fname := item.GetFilename()
			assert.Equal(t, tt.expected, fname)
		})
	}
}

func TestDownloadAudioIfRequestError(t *testing.T) {
	var ts *httptest.Server
	ts = httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		ts.CloseClientConnections()
	}))

	defer ts.Close()

	item := Item{Enclosure: Enclosure{URL: ts.URL}}
	got, err := item.DownloadAudio(time.Minute)

	assert.Nil(t, got)
	assert.EqualError(t, err, fmt.Sprintf("can't download %s: Get %q: EOF", ts.URL, ts.URL))
}

func TestDownloadAudio(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Length", "4")
		fmt.Fprint(w, "abcd")
	}))
	defer ts.Close()

	item := Item{Enclosure: Enclosure{URL: ts.URL}}
	got, err := item.DownloadAudio(time.Minute)

	assert.NotNil(t, got)
	assert.Nil(t, err)
}
