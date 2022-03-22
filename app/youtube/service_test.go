package youtube

import (
	"context"
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
	}

	svc := Service{
		Channels:       []ChannelInfo{{ID: "channel1", Name: "name1"}, {ID: "channel2", Name: "name2"}},
		Downloader:     downloader,
		ChannelService: chans,
		Store:          store,
		CheckDuration:  time.Millisecond * 500,
		KeepPerChannel: 10,
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
	require.True(t, strings.HasPrefix(store.SaveCalls()[0].Entry.File, "/tmp/"))
	require.True(t, strings.HasSuffix(store.SaveCalls()[0].Entry.File, ".mp3"))
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
