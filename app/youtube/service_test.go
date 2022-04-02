package youtube

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"

	ytfeed "github.com/umputun/feed-master/app/youtube/feed"
	"github.com/umputun/feed-master/app/youtube/store"

	"github.com/umputun/feed-master/app/youtube/mocks"
)

func TestService_Do(t *testing.T) {

	chans := &mocks.ChannelServiceMock{
		GetFunc: func(ctx context.Context, chanID string, feedType ytfeed.Type) ([]ytfeed.Entry, error) {
			return []ytfeed.Entry{
				{ChannelID: chanID, VideoID: "vid1", Title: "title1", Published: time.Now()},
				{ChannelID: chanID, VideoID: "vid2", Title: "title2", Published: time.Now()},
				{ChannelID: chanID, VideoID: "vid2", Title: "title2", Published: time.Now()}, // duplicate
			}, nil
		},
	}
	downloader := &mocks.DownloaderServiceMock{
		GetFunc: func(ctx context.Context, id string, fname string) (string, error) {
			return "/tmp/" + fname + ".mp3", nil
		},
	}

	tmpfile := filepath.Join(os.TempDir(), "test.db")
	defer os.Remove(tmpfile)

	db, err := bolt.Open(tmpfile, 0o600, &bolt.Options{Timeout: 1 * time.Second})
	require.NoError(t, err)
	boltStore := &store.BoltDB{DB: db}
	svc := Service{
		Feeds: []FeedInfo{
			{ID: "channel1", Name: "name1", Type: ytfeed.FTChannel},
			{ID: "channel2", Name: "name2", Type: ytfeed.FTPlaylist},
		},
		Downloader:     downloader,
		ChannelService: chans,
		Store:          boltStore,
		CheckDuration:  time.Millisecond * 500,
		KeepPerChannel: 10,
		RSSFileStore:   RSSFileStore{Enabled: true, Location: "/tmp"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*900)
	defer cancel()

	err = svc.Do(ctx)
	assert.EqualError(t, err, "context deadline exceeded")

	require.Equal(t, 4, len(chans.GetCalls()))
	assert.Equal(t, "channel1", chans.GetCalls()[0].ChanID)
	assert.Equal(t, ytfeed.FTChannel, chans.GetCalls()[0].FeedType)
	assert.Equal(t, "channel2", chans.GetCalls()[1].ChanID)
	assert.Equal(t, ytfeed.FTPlaylist, chans.GetCalls()[1].FeedType)
	assert.Equal(t, "channel1", chans.GetCalls()[2].ChanID)
	assert.Equal(t, "channel2", chans.GetCalls()[3].ChanID)

	res, err := boltStore.Load("channel1", 10)
	require.NoError(t, err)
	assert.Equal(t, 2, len(res), "two entries for channel1, skipped duplicate")
	assert.Equal(t, "vid2", res[0].VideoID)
	assert.Equal(t, "vid1", res[1].VideoID)

	res, err = boltStore.Load("channel2", 10)
	require.NoError(t, err)
	assert.Equal(t, 2, len(res), "two entries for channel1, skipped duplicate")
	assert.Equal(t, "vid2", res[0].VideoID)
	assert.Equal(t, "vid1", res[1].VideoID)

	require.Equal(t, 4, len(downloader.GetCalls()))
	require.Equal(t, "vid1", downloader.GetCalls()[0].ID)
	require.True(t, downloader.GetCalls()[0].Fname != "")

	rssData, err := os.ReadFile("/tmp/channel1.xml")
	require.NoError(t, err)
	t.Logf("%s", string(rssData))
	assert.Contains(t, string(rssData), "<guid>channel1::vid1</guid>")
	assert.Contains(t, string(rssData), "<guid>channel1::vid2</guid>")

	rssData, err = os.ReadFile("/tmp/channel2.xml")
	require.NoError(t, err)
	assert.Contains(t, string(rssData), "<guid>channel2::vid1</guid>")
	assert.Contains(t, string(rssData), "<guid>channel2::vid2</guid>")
}

// nolint:dupl // test if very similar to TestService_RSSFeed
func TestService_RSSFeed(t *testing.T) {
	storeSvc := &mocks.StoreServiceMock{
		LoadFunc: func(channelID string, max int) ([]ytfeed.Entry, error) {
			res := []ytfeed.Entry{
				{ChannelID: "channel1", VideoID: "vid1", Title: "title1", File: "/tmp/file1.mp3"},
				{ChannelID: "channel1", VideoID: "vid2", Title: "title2", File: "/tmp/file2.mp3"},
			}
			res[0].Link.Href = "http://example.com/v1"
			res[1].Link.Href = "http://example.com/v2"
			res[0].Author.URI = "http://example.com/c1"
			res[0].Media.Thumbnail.URL = "http://example.com/thumb.jpg"
			return res, nil
		},
	}

	svc := Service{
		Feeds: []FeedInfo{
			{ID: "channel1", Name: "name1", Type: ytfeed.FTChannel},
			{ID: "channel2", Name: "name2", Type: ytfeed.FTPlaylist},
		},
		Store:          storeSvc,
		RootURL:        "http://localhost:8080/yt",
		KeepPerChannel: 10,
	}

	res, err := svc.RSSFeed(FeedInfo{ID: "channel1", Name: "name1", Type: ytfeed.FTChannel})
	require.NoError(t, err)
	t.Logf("%v", res)

	assert.Contains(t, res, `<enclosure url="http://localhost:8080/yt/file1.mp3"`)
	assert.Contains(t, res, `<enclosure url="http://localhost:8080/yt/file1.mp3"`)
	assert.Contains(t, res, `<guid>channel1::vid1</guid>`)
	assert.Contains(t, res, `<guid>channel1::vid2</guid>`)
	assert.Contains(t, res, `<link>http://example.com/v1</link>`)
	assert.Contains(t, res, `<link>http://example.com/v2</link>`)
	assert.Contains(t, res, `<link>http://example.com/c1</link>`)
	assert.Contains(t, res, `<itunes:image href="http://example.com/thumb.jpg"></itunes:image>`)
	assert.Contains(t, res, `<media:thumbnail url="http://example.com/thumb.jpg"></media:thumbnail>`)
}

// nolint:dupl // test if very similar to TestService_RSSFeed
func TestService_RSSFeedPlayList(t *testing.T) {
	storeSvc := &mocks.StoreServiceMock{
		LoadFunc: func(channelID string, max int) ([]ytfeed.Entry, error) {
			res := []ytfeed.Entry{
				{ChannelID: "channel1", VideoID: "vid1", Title: "title1", File: "/tmp/file1.mp3"},
				{ChannelID: "channel1", VideoID: "vid2", Title: "title2", File: "/tmp/file2.mp3"},
			}
			res[0].Link.Href = "http://example.com/v1"
			res[1].Link.Href = "http://example.com/v2"
			res[0].Author.URI = "http://example.com/c1"
			return res, nil
		},
	}

	svc := Service{
		Feeds: []FeedInfo{
			{ID: "channel1", Name: "name1", Type: ytfeed.FTPlaylist},
			{ID: "channel2", Name: "name2", Type: ytfeed.FTPlaylist},
		},
		Store:          storeSvc,
		RootURL:        "http://localhost:8080/yt",
		KeepPerChannel: 10,
	}

	res, err := svc.RSSFeed(FeedInfo{ID: "channel1", Name: "name1", Type: ytfeed.FTPlaylist})
	require.NoError(t, err)
	t.Logf("%v", res)

	assert.Contains(t, res, `<enclosure url="http://localhost:8080/yt/file1.mp3"`)
	assert.Contains(t, res, `<enclosure url="http://localhost:8080/yt/file1.mp3"`)
	assert.Contains(t, res, `<guid>channel1::vid1</guid>`)
	assert.Contains(t, res, `<guid>channel1::vid2</guid>`)
	assert.Contains(t, res, `<link>http://example.com/v1</link>`)
	assert.Contains(t, res, `<link>http://example.com/v2</link>`)
	assert.Contains(t, res, `<link>https://www.youtube.com/playlist?list=channel1</link>`)
}

func TestService_makeFileName(t *testing.T) {

	tbl := []struct {
		entry ytfeed.Entry
		res   string
	}{
		{
			entry: ytfeed.Entry{ChannelID: "channel1", VideoID: "vid1", Title: "title1"},
			res:   "e4650bb3d770eed60faad7ffbed5f33ffb1b89fa",
		},
		{
			entry: ytfeed.Entry{ChannelID: "channel1", VideoID: "vid2", Title: "title2"},
			res:   "4308c33c7ddb107c2d0c13a905e4c6962001bab4",
		},
		{
			entry: ytfeed.Entry{ChannelID: "channel2", VideoID: "vid1", Title: "title1"},
			res:   "3be877c750abb87daee80c005fe87e7a3f824fed",
		},
		{
			entry: ytfeed.Entry{ChannelID: "channel2", VideoID: "vid2", Title: "title2"},
			res:   "648f79b3a05ececb8a37600aa0aee332f0374e01",
		},
		{
			entry: ytfeed.Entry{ChannelID: "channel2", VideoID: "vid2", Title: "title2"},
			res:   "648f79b3a05ececb8a37600aa0aee332f0374e01",
		},
	}

	svc := Service{}
	for i, tt := range tbl {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert.Equal(t, tt.res, svc.makeFileName(tt.entry))
		})
	}

}
