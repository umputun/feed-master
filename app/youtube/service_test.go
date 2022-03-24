package youtube

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/feed-master/app/youtube/channel"
	"github.com/umputun/feed-master/app/youtube/mocks"
)

func TestService_Do(t *testing.T) {

	chans := &mocks.ChannelServiceMock{
		GetFunc: func(ctx context.Context, chanID string) ([]channel.Entry, error) {
			return []channel.Entry{
				{ChannelID: chanID, VideoID: "vid1", Title: "title1"},
				{ChannelID: chanID, VideoID: "vid2", Title: "title2"},
			}, nil
		},
	}
	downloader := &mocks.DownloaderServiceMock{
		GetFunc: func(ctx context.Context, id string, fname string) (string, error) {
			return "/tmp/" + fname + ".mp3", nil
		},
	}
	store := &mocks.StoreServiceMock{
		ExistFunc: func(entry channel.Entry) (bool, error) {
			if entry.VideoID == "vid2" {
				return true, nil
			}
			return false, nil
		},
		SaveFunc: func(entry channel.Entry) (bool, error) {
			return true, nil
		},

		RemoveOldFunc: func(channelID string, keep int) ([]string, error) {
			return []string{"/tmp/blah.mp3"}, nil
		},
		LoadFunc: func(channelID string, max int) ([]channel.Entry, error) {
			return []channel.Entry{
				{ChannelID: channelID, VideoID: "vid1", Title: "title1"},
				{ChannelID: channelID, VideoID: "vid2", Title: "title2"},
			}, nil
		},
	}

	svc := Service{
		Channels:       []ChannelInfo{{ID: "channel1", Name: "name1"}, {ID: "channel2", Name: "name2"}},
		Downloader:     downloader,
		ChannelService: chans,
		Store:          store,
		CheckDuration:  time.Millisecond * 500,
		KeepPerChannel: 10,
		RSSFileStore:   RSSFileStore{Enabled: true, Location: "/tmp"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*900)
	defer cancel()

	err := svc.Do(ctx)
	assert.EqualError(t, err, "context deadline exceeded")

	require.Equal(t, 4, len(chans.GetCalls()))
	assert.Equal(t, "channel1", chans.GetCalls()[0].ChanID)
	assert.Equal(t, "channel2", chans.GetCalls()[1].ChanID)
	assert.Equal(t, "channel1", chans.GetCalls()[2].ChanID)
	assert.Equal(t, "channel2", chans.GetCalls()[3].ChanID)

	require.Equal(t, 8, len(store.ExistCalls()))
	require.Equal(t, "channel1", store.ExistCalls()[0].Entry.ChannelID)
	require.Equal(t, "channel1", store.ExistCalls()[1].Entry.ChannelID)
	require.Equal(t, "channel2", store.ExistCalls()[2].Entry.ChannelID)
	require.Equal(t, "channel2", store.ExistCalls()[3].Entry.ChannelID)

	require.Equal(t, 4, len(downloader.GetCalls()))
	require.Equal(t, "vid1", downloader.GetCalls()[0].ID)
	require.True(t, downloader.GetCalls()[0].Fname != "")

	require.Equal(t, 4, len(store.SaveCalls()))
	require.Equal(t, "channel1", store.SaveCalls()[0].Entry.ChannelID)
	require.Equal(t, "vid1", store.SaveCalls()[0].Entry.VideoID)
	require.Equal(t, "name1: title1", store.SaveCalls()[0].Entry.Title)
	require.True(t, strings.HasPrefix(store.SaveCalls()[0].Entry.File, "/tmp/"))
	require.True(t, strings.HasSuffix(store.SaveCalls()[0].Entry.File, ".mp3"))

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

func TestService_RSSFeed(t *testing.T) {
	store := &mocks.StoreServiceMock{
		LoadFunc: func(channelID string, max int) ([]channel.Entry, error) {
			return []channel.Entry{
				{ChannelID: "channel1", VideoID: "vid1", Title: "title1", File: "/tmp/file1.mp3"},
				{ChannelID: "channel1", VideoID: "vid2", Title: "title2", File: "/tmp/file2.mp3"},
			}, nil
		},
	}

	svc := Service{
		Channels:       []ChannelInfo{{ID: "channel1", Name: "name1"}, {ID: "channel2", Name: "name2"}},
		Store:          store,
		RootURL:        "http://localhost:8080/yt",
		KeepPerChannel: 10,
	}

	res, err := svc.RSSFeed(ChannelInfo{ID: "channel1", Name: "name1"})
	require.NoError(t, err)
	t.Logf("%v", res)

	assert.Contains(t, res, `<enclosure url="http://localhost:8080/yt/file1.mp3"`)
	assert.Contains(t, res, `<enclosure url="http://localhost:8080/yt/file1.mp3"`)
	assert.Contains(t, res, `<guid>channel1::vid1</guid>`)
	assert.Contains(t, res, `<guid>channel1::vid2</guid>`)
}

func TestService_makeFileName(t *testing.T) {

	tbl := []struct {
		entry channel.Entry
		res   string
	}{
		{
			entry: channel.Entry{ChannelID: "channel1", VideoID: "vid1", Title: "title1"},
			res:   "e4650bb3d770eed60faad7ffbed5f33ffb1b89fa",
		},
		{
			entry: channel.Entry{ChannelID: "channel1", VideoID: "vid2", Title: "title2"},
			res:   "4308c33c7ddb107c2d0c13a905e4c6962001bab4",
		},
		{
			entry: channel.Entry{ChannelID: "channel2", VideoID: "vid1", Title: "title1"},
			res:   "3be877c750abb87daee80c005fe87e7a3f824fed",
		},
		{
			entry: channel.Entry{ChannelID: "channel2", VideoID: "vid2", Title: "title2"},
			res:   "648f79b3a05ececb8a37600aa0aee332f0374e01",
		},
		{
			entry: channel.Entry{ChannelID: "channel2", VideoID: "vid2", Title: "title2"},
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
