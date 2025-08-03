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

	tempDir := t.TempDir()
	shortVideo := filepath.Join(tempDir, "122b672d10e77708b51c041f852615dc0eedf354.mp3")
	chans := &mocks.ChannelServiceMock{
		GetFunc: func(_ context.Context, chanID string, _ ytfeed.Type) ([]ytfeed.Entry, error) {
			return []ytfeed.Entry{
				{ChannelID: chanID, VideoID: "vid1", Title: "title1", Published: time.Now()},
				{ChannelID: chanID, VideoID: "vid2", Title: "title2", Published: time.Now()},
				{ChannelID: chanID, VideoID: "vid2", Title: "title2", Published: time.Now()},               // duplicate
				{ChannelID: chanID, VideoID: "vid3", Title: "title3", Published: time.Now(), Duration: 40}, // short
			}, nil
		},
	}
	downloader := &mocks.DownloaderServiceMock{
		GetFunc: func(_ context.Context, _ string, fname string) (string, error) {
			fpath := filepath.Join(tempDir, fname+".mp3")
			_, err := os.Create(fpath) // nolint
			require.NoError(t, err)
			return fpath, nil
		},
	}

	duration := &mocks.DurationServiceMock{
		FileFunc: func(fname string) int {
			if fname == shortVideo {
				return 30
			}
			return 1234
		},
	}

	tmpfile := filepath.Join(tempDir, "test.db")
	defer os.Remove(tmpfile)

	db, err := bolt.Open(tmpfile, 0o600, &bolt.Options{Timeout: 1 * time.Second})
	require.NoError(t, err)
	boltStore := &store.BoltDB{DB: db}
	svc := Service{
		Feeds: []FeedInfo{
			{ID: "channel1", Name: "name1", Type: ytfeed.FTChannel},
			{ID: "channel2", Name: "name2", Type: ytfeed.FTPlaylist},
		},
		Downloader:      downloader,
		ChannelService:  chans,
		Store:           boltStore,
		CheckDuration:   time.Millisecond * 500,
		KeepPerChannel:  10,
		RSSFileStore:    RSSFileStore{Enabled: true, Location: tempDir},
		DurationService: duration,
		SkipShorts:      time.Second * 60,
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
	assert.Equal(t, 3, len(res), "three entries for channel2, skipped duplicate")
	assert.Equal(t, "vid3", res[0].VideoID)
	assert.Equal(t, "vid2", res[1].VideoID)
	assert.Equal(t, "vid1", res[2].VideoID)

	require.Equal(t, 6, len(downloader.GetCalls()))
	require.Equal(t, "vid1", downloader.GetCalls()[0].ID)
	require.True(t, downloader.GetCalls()[0].Fname != "")

	rssData, err := os.ReadFile(tempDir + "/channel1.xml") // nolint
	require.NoError(t, err)
	t.Logf("%s", string(rssData))
	assert.Contains(t, string(rssData), "<guid>channel1::vid1</guid>")
	assert.Contains(t, string(rssData), "<guid>channel1::vid2</guid>")
	assert.Contains(t, string(rssData), "<itunes:duration>1234</itunes:duration>")

	rssData, err = os.ReadFile(tempDir + "/channel2.xml") // nolint
	require.NoError(t, err)
	assert.Contains(t, string(rssData), "<guid>channel2::vid1</guid>")
	assert.Contains(t, string(rssData), "<guid>channel2::vid2</guid>")
	assert.Contains(t, string(rssData), "<itunes:duration>1234</itunes:duration>")

	t.Logf("%v", duration.FileCalls())
	// durationService.File called 11 times: 5 in Service.update(), 6 in Service.isShort()
	require.Equal(t, 11, len(duration.FileCalls()))
	assert.Equal(t, filepath.Join(tempDir, "e4650bb3d770eed60faad7ffbed5f33ffb1b89fa.mp3"), duration.FileCalls()[0].Fname)
	assert.Equal(t, filepath.Join(tempDir, "4308c33c7ddb107c2d0c13a905e4c6962001bab4.mp3"), duration.FileCalls()[2].Fname)
	assert.Equal(t, filepath.Join(tempDir, "122b672d10e77708b51c041f852615dc0eedf354.mp3"), duration.FileCalls()[4].Fname)
	assert.Equal(t, filepath.Join(tempDir, "3be877c750abb87daee80c005fe87e7a3f824fed.mp3"), duration.FileCalls()[6].Fname)
	assert.Equal(t, filepath.Join(tempDir, "648f79b3a05ececb8a37600aa0aee332f0374e01.mp3"), duration.FileCalls()[8].Fname)
	assert.NoFileExists(t, shortVideo, "short video should be removed")
	assert.FileExists(t, filepath.Join(tempDir, "e4650bb3d770eed60faad7ffbed5f33ffb1b89fa.mp3"), "non short video should exist")
}

// nolint:dupl // test if very similar to TestService_RSSFeed
func TestService_DoIsAllowedFilter(t *testing.T) {

	chans := &mocks.ChannelServiceMock{
		GetFunc: func(_ context.Context, chanID string, _ ytfeed.Type) ([]ytfeed.Entry, error) {
			return []ytfeed.Entry{
				{ChannelID: chanID, VideoID: "vid1", Title: "Prefix1: title1", Published: time.Now()},
				{ChannelID: chanID, VideoID: "vid2", Title: "Prefix2: title2", Published: time.Now()},
				{ChannelID: chanID, VideoID: "vid3", Title: "Prefix2: title3", Published: time.Now()},
			}, nil
		},
	}
	downloader := &mocks.DownloaderServiceMock{
		GetFunc: func(_ context.Context, _ string, fname string) (string, error) {
			return "/tmp/" + fname + ".mp3", nil
		},
	}

	duration := &mocks.DurationServiceMock{
		FileFunc: func(string) int {
			return 1234
		},
	}

	tmpfile := filepath.Join(os.TempDir(), "test.db")
	defer os.Remove(tmpfile)

	db, err := bolt.Open(tmpfile, 0o600, &bolt.Options{Timeout: 5 * time.Second})
	require.NoError(t, err)
	boltStore := &store.BoltDB{DB: db}
	svc := Service{
		Feeds: []FeedInfo{
			{ID: "channel1", Name: "name1", Type: ytfeed.FTChannel, Filter: FeedFilter{Include: "Prefix2", Exclude: "title3"}},
			{ID: "channel2", Name: "name2", Type: ytfeed.FTChannel, Filter: FeedFilter{Include: "^\\w{7}:", Exclude: "\\w+3$"}},
		},
		Downloader:      downloader,
		ChannelService:  chans,
		Store:           boltStore,
		CheckDuration:   time.Millisecond * 500,
		KeepPerChannel:  10,
		RSSFileStore:    RSSFileStore{Enabled: true, Location: "/tmp"},
		DurationService: duration,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*900)
	defer cancel()

	err = svc.Do(ctx)
	assert.EqualError(t, err, "context deadline exceeded")

	require.Equal(t, 4, len(chans.GetCalls()))
	assert.Equal(t, "channel1", chans.GetCalls()[0].ChanID)
	assert.Equal(t, ytfeed.FTChannel, chans.GetCalls()[0].FeedType)
	assert.Equal(t, "channel2", chans.GetCalls()[1].ChanID)
	assert.Equal(t, ytfeed.FTChannel, chans.GetCalls()[1].FeedType)
	assert.Equal(t, "channel1", chans.GetCalls()[2].ChanID)
	assert.Equal(t, "channel2", chans.GetCalls()[3].ChanID)

	res, err := boltStore.Load("channel1", 10)
	require.NoError(t, err)
	assert.Equal(t, 1, len(res), "one entry for channel1, skipped irrelevant ones")
	assert.Equal(t, "vid2", res[0].VideoID)

	res, err = boltStore.Load("channel2", 10)
	require.NoError(t, err)
	assert.Equal(t, 2, len(res), "two entries for channel2, skipped irrelevant one")
	assert.Equal(t, "vid2", res[0].VideoID)
	assert.Equal(t, "vid1", res[1].VideoID)

	require.Equal(t, 3, len(downloader.GetCalls()))
	require.Equal(t, "vid2", downloader.GetCalls()[0].ID)
	require.Equal(t, "vid1", downloader.GetCalls()[1].ID)
	require.Equal(t, "vid2", downloader.GetCalls()[2].ID)
	require.True(t, downloader.GetCalls()[0].Fname != "")

	rssData, err := os.ReadFile("/tmp/channel1.xml")
	require.NoError(t, err)
	t.Logf("%s", string(rssData))
	assert.Contains(t, string(rssData), "<guid>channel1::vid2</guid>")
	assert.Contains(t, string(rssData), "<itunes:duration>1234</itunes:duration>")

	rssData, err = os.ReadFile("/tmp/channel2.xml")
	require.NoError(t, err)
	t.Logf("%s", string(rssData))
	assert.Contains(t, string(rssData), "<guid>channel2::vid2</guid>")
	assert.Contains(t, string(rssData), "<guid>channel2::vid1</guid>")
	assert.Contains(t, string(rssData), "<itunes:duration>1234</itunes:duration>")

	require.Equal(t, 3, len(duration.FileCalls()))
	assert.Equal(t, "/tmp/4308c33c7ddb107c2d0c13a905e4c6962001bab4.mp3", duration.FileCalls()[0].Fname)
	assert.Equal(t, "/tmp/3be877c750abb87daee80c005fe87e7a3f824fed.mp3", duration.FileCalls()[1].Fname)
	assert.Equal(t, "/tmp/648f79b3a05ececb8a37600aa0aee332f0374e01.mp3", duration.FileCalls()[2].Fname)
}

// nolint:dupl // test if very similar to TestService_RSSFeed
func TestService_RSSFeed(t *testing.T) {
	storeSvc := &mocks.StoreServiceMock{
		LoadFunc: func(string, int) ([]ytfeed.Entry, error) {
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
		SkipShorts:     time.Second * 60,
	}

	res, err := svc.RSSFeed(FeedInfo{ID: "channel1", Name: "name1", Type: ytfeed.FTChannel})
	require.NoError(t, err)
	t.Logf("%v", res)

	assert.Contains(t, res, `<rss version="2.0" xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd" xmlns:media="http://search.yahoo.com/mrss/">`)
	assert.Contains(t, res, `<enclosure url="http://localhost:8080/yt/file1.mp3"`)
	assert.Contains(t, res, `<enclosure url="http://localhost:8080/yt/file1.mp3"`)
	assert.Contains(t, res, `<guid>channel1::vid1</guid>`)
	assert.Contains(t, res, `<guid>channel1::vid2</guid>`)
	assert.NotContains(t, res, `<guid>channel1::vid3</guid>`, "skipped short video")
	assert.Contains(t, res, `<link>http://example.com/v1</link>`)
	assert.Contains(t, res, `<link>http://example.com/v2</link>`)
	assert.Contains(t, res, `<link>http://example.com/c1</link>`)
	assert.Contains(t, res, `<itunes:image href="http://example.com/thumb.jpg"></itunes:image>`)
	assert.Contains(t, res, `<media:thumbnail url="http://example.com/thumb.jpg"></media:thumbnail>`)

}

// nolint:dupl // test if very similar to TestService_RSSFeed
func TestService_RSSFeedPlayList(t *testing.T) {
	storeSvc := &mocks.StoreServiceMock{
		LoadFunc: func(string, int) ([]ytfeed.Entry, error) {
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

func TestService_update(t *testing.T) {

	duration := &mocks.DurationServiceMock{
		FileFunc: func(string) int {
			return 1234
		},
	}

	svc := Service{DurationService: duration}

	{ // update with reset pub time
		inpEntry := ytfeed.Entry{
			ChannelID: "chan1",
			VideoID:   "vid1",
			Updated:   time.Now().Add(time.Minute * -1),
			Published: time.Now().Add(time.Hour * -1),
			Title:     "something",
		}

		res := svc.update(inpEntry, "/tmp/audio.mp3", FeedInfo{ID: "f1", Name: "feed1"})
		t.Logf("%+v", res)
		assert.Equal(t, 1234, res.Duration)
		assert.True(t, time.Since(res.Published) < time.Second, "published time was reset")
		assert.Equal(t, "feed1: something", res.Title)
	}

	{ // update with altering title for dedup
		inpEntry := ytfeed.Entry{
			ChannelID: "chan1",
			VideoID:   "vid1",
			Updated:   time.Now().Add(time.Minute * -1),
			Published: time.Now().Add(time.Hour * -1),
			Title:     `Сергей Пархоменко на канале “Живой Гвоздь” в программме “Персонально ваш”. 06.04.2022`,
		}
		res := svc.update(inpEntry, "/tmp/audio.mp3", FeedInfo{ID: "f1", Name: "Сергей Пархоменко"})
		t.Logf("%+v", res)
		assert.Equal(t, 1234, res.Duration)
		assert.True(t, time.Since(res.Published) < time.Second, "published time was reset")
		assert.Equal(t, "Сергей Пархоменко на канале “Живой Гвоздь” в программме “Персонально ваш”. 06.04.2022", res.Title)
	}

}

func TestService_totalEntriesToKeep(t *testing.T) {
	svc := Service{
		Feeds: []FeedInfo{
			{ID: "channel1", Name: "name1", Type: ytfeed.FTChannel, Keep: 5},
			{ID: "channel2", Name: "name2", Type: ytfeed.FTPlaylist},
		},
		RootURL:        "http://localhost:8080/yt",
		KeepPerChannel: 10,
	}

	assert.Equal(t, 15, svc.totalEntriesToKeep())
}

func TestService_countAllEntries(t *testing.T) {

	storeSvc := &mocks.StoreServiceMock{
		LoadFunc: func(channelID string, _ int) ([]ytfeed.Entry, error) {
			switch channelID {
			case "channel1":
				return []ytfeed.Entry{{}, {}, {}}, nil
			case "channel2":
				return []ytfeed.Entry{{}, {}}, nil
			default:
				t.Fatalf("unexpected channelID: %s", channelID)
			}
			return nil, nil
		},
	}

	svc := Service{
		Feeds: []FeedInfo{
			{ID: "channel1", Name: "name1", Type: ytfeed.FTChannel, Keep: 5},
			{ID: "channel2", Name: "name2", Type: ytfeed.FTPlaylist},
		},
		Store:          storeSvc,
		KeepPerChannel: 10,
	}

	assert.Equal(t, 5, svc.countAllEntries())
	assert.Equal(t, 2, len(storeSvc.LoadCalls()))
	assert.Equal(t, 5, storeSvc.LoadCalls()[0].Max)
	assert.Equal(t, 10, storeSvc.LoadCalls()[1].Max)
}

func TestService_oldestEntry(t *testing.T) {
	dt := time.Date(2022, 4, 11, 11, 35, 17, 0, time.UTC)
	storeSvc := &mocks.StoreServiceMock{
		LoadFunc: func(channelID string, _ int) ([]ytfeed.Entry, error) {
			switch channelID {
			case "channel1":
				return []ytfeed.Entry{
					{Title: "t1", Published: dt.Add(4 * time.Hour)},
					{Title: "t2", Published: dt.Add(3 * time.Hour)},
					{Title: "t3", Published: dt.Add(2 * time.Hour)},
				}, nil
			case "channel2":

				return []ytfeed.Entry{
					{Title: "t21", Published: dt.Add(5 * time.Hour)},
					{Title: "t22", Published: dt.Add(2 * time.Hour)},
					{Title: "t23", Published: dt.Add(1 * time.Hour)},
				}, nil
			default:
				t.Fatalf("unexpected channelID: %s", channelID)
			}
			return nil, nil
		},
	}

	{
		svc := Service{
			Feeds: []FeedInfo{
				{ID: "channel1", Name: "name1", Type: ytfeed.FTChannel, Keep: 5},
				{ID: "channel2", Name: "name2", Type: ytfeed.FTPlaylist},
			},
			Store:          storeSvc,
			KeepPerChannel: 10,
		}

		res := svc.oldestEntry()
		assert.Equal(t, "t23", res.Title)
	}

	assert.Equal(t, 2, len(storeSvc.LoadCalls()))
	assert.Equal(t, 5, storeSvc.LoadCalls()[0].Max)
	assert.Equal(t, 10, storeSvc.LoadCalls()[1].Max)
}

func TestService_newestEntry(t *testing.T) {
	dt := time.Date(2022, 4, 11, 11, 35, 17, 0, time.UTC)
	storeSvc := &mocks.StoreServiceMock{
		LoadFunc: func(channelID string, _ int) ([]ytfeed.Entry, error) {
			switch channelID {
			case "channel1":
				return []ytfeed.Entry{
					{Title: "t1", Published: dt.Add(4 * time.Hour)},
					{Title: "t2", Published: dt.Add(3 * time.Hour)},
					{Title: "t3", Published: dt.Add(2 * time.Hour)},
				}, nil
			case "channel2":

				return []ytfeed.Entry{
					{Title: "t21", Published: dt.Add(5 * time.Hour)},
					{Title: "t22", Published: dt.Add(2 * time.Hour)},
					{Title: "t23", Published: dt.Add(1 * time.Hour)},
				}, nil
			default:
				t.Fatalf("unexpected channelID: %s", channelID)
			}
			return nil, nil
		},
	}

	{
		svc := Service{
			Feeds: []FeedInfo{
				{ID: "channel1", Name: "name1", Type: ytfeed.FTChannel, Keep: 5},
				{ID: "channel2", Name: "name2", Type: ytfeed.FTPlaylist},
			},
			Store:          storeSvc,
			KeepPerChannel: 10,
		}

		res := svc.newestEntry()
		assert.Equal(t, "t21", res.Title)
	}

	assert.Equal(t, 2, len(storeSvc.LoadCalls()))
	assert.Equal(t, 1, storeSvc.LoadCalls()[0].Max)
	assert.Equal(t, 1, storeSvc.LoadCalls()[1].Max)
}
