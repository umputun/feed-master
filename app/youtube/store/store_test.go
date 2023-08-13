package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"

	"github.com/umputun/feed-master/app/youtube/feed"
)

func TestStore_SaveAndLoad(t *testing.T) {
	tmpfile := filepath.Join(os.TempDir(), "test.db")
	defer os.Remove(tmpfile)

	db, err := bolt.Open(tmpfile, 0o600, &bolt.Options{Timeout: 5 * time.Second})
	require.NoError(t, err)

	s := BoltDB{DB: db}

	entry := feed.Entry{
		ChannelID: "chan1",
		VideoID:   "vid1",
		Title:     "title1",
		Published: time.Date(2022, time.March, 21, 16, 45, 22, 0, time.UTC),
	}

	created, err := s.Save(entry)
	require.NoError(t, err)
	assert.True(t, created)

	created, err = s.Save(entry)
	require.NoError(t, err)
	assert.False(t, created)

	res, err := s.Load("chan1", 100)
	require.NoError(t, err)
	assert.Equal(t, 1, len(res))
	assert.Equal(t, "vid1", res[0].VideoID)

	entry2 := feed.Entry{
		ChannelID: "chan1",
		VideoID:   "vid2",
		Title:     "title2",
		Published: time.Date(2022, time.March, 21, 17, 45, 22, 0, time.UTC),
	}
	created, err = s.Save(entry2)
	require.NoError(t, err)
	assert.True(t, created)

	res, err = s.Load("chan1", 100)
	require.NoError(t, err)
	assert.Equal(t, 2, len(res))
	assert.Equal(t, "vid2", res[0].VideoID)

	res, err = s.Load("chan1", 1)
	require.NoError(t, err)
	assert.Equal(t, 1, len(res))
	assert.Equal(t, "vid2", res[0].VideoID)
}

func TestStore_Remove(t *testing.T) {
	tmpfile := filepath.Join(os.TempDir(), "test.db")
	defer os.Remove(tmpfile)

	db, err := bolt.Open(tmpfile, 0o600, &bolt.Options{Timeout: 1 * time.Second})
	require.NoError(t, err)

	s := BoltDB{DB: db}

	entry := feed.Entry{
		ChannelID: "chan1",
		VideoID:   "vid1",
		Title:     "title1",
		Published: time.Date(2022, time.March, 21, 16, 45, 22, 0, time.UTC),
	}

	created, err := s.Save(entry)
	require.NoError(t, err)
	assert.True(t, created)

	entry2 := feed.Entry{
		ChannelID: "chan1",
		VideoID:   "vid2",
		Title:     "title2",
		Published: time.Date(2022, time.March, 21, 17, 45, 22, 0, time.UTC),
	}
	created, err = s.Save(entry2)
	require.NoError(t, err)
	assert.True(t, created)

	res, err := s.Load("chan1", 100)
	require.NoError(t, err)
	assert.Equal(t, 2, len(res))
	assert.Equal(t, "vid2", res[0].VideoID)

	err = s.Remove(feed.Entry{ChannelID: "chan1", VideoID: "vid2"})
	require.NoError(t, err)
	res, err = s.Load("chan1", 10)
	require.NoError(t, err)
	assert.Equal(t, 1, len(res))
	assert.Equal(t, "vid1", res[0].VideoID)
}

func TestStore_Exist(t *testing.T) {
	tmpfile := filepath.Join(os.TempDir(), "test.db")
	defer os.Remove(tmpfile)

	db, err := bolt.Open(tmpfile, 0o600, &bolt.Options{Timeout: 1 * time.Second})
	require.NoError(t, err)

	s := BoltDB{DB: db}

	entry := feed.Entry{
		ChannelID: "chan1",
		VideoID:   "vid1",
		Title:     "title1",
		Published: time.Date(2022, time.March, 21, 16, 45, 22, 0, time.UTC),
	}

	created, err := s.Save(entry)
	require.NoError(t, err)
	assert.True(t, created)

	ok, err := s.Exist(entry)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = s.Exist(feed.Entry{ChannelID: "chan2", VideoID: "vid2"})
	require.NoError(t, err)
	assert.False(t, ok)

}

func TestBoldDB_RemoveOld(t *testing.T) {
	tmpfile := filepath.Join(os.TempDir(), "test.db")
	defer os.Remove(tmpfile)

	db, err := bolt.Open(tmpfile, 0o600, &bolt.Options{Timeout: 1 * time.Second})
	require.NoError(t, err)

	s := BoltDB{DB: db}
	{
		entry := feed.Entry{
			ChannelID: "chan1",
			VideoID:   "vid1",
			Title:     "title1",
			Published: time.Date(2022, time.March, 21, 16, 45, 22, 0, time.UTC),
			File:      "f1",
		}
		created, e := s.Save(entry)
		require.NoError(t, e)
		assert.True(t, created)
	}
	{
		entry := feed.Entry{
			ChannelID: "chan1",
			VideoID:   "vid2",
			Title:     "title2",
			Published: time.Date(2022, time.March, 21, 17, 45, 22, 0, time.UTC),
			File:      "f2",
		}
		created, e := s.Save(entry)
		require.NoError(t, e)
		assert.True(t, created)
	}
	{
		entry := feed.Entry{
			ChannelID: "chan1",
			VideoID:   "vid3",
			Title:     "title3",
			Published: time.Date(2022, time.March, 21, 18, 45, 22, 0, time.UTC),
			File:      "f3",
		}
		created, e := s.Save(entry)
		require.NoError(t, e)
		assert.True(t, created)
	}

	res, err := s.RemoveOld("chan1", 1)
	require.NoError(t, err)
	assert.Equal(t, []string{"f2", "f1"}, res)
}

func TestBoltDB_SetProcessed(t *testing.T) {
	tmpfile := filepath.Join(os.TempDir(), "test.db")
	defer os.Remove(tmpfile)

	db, err := bolt.Open(tmpfile, 0o600, &bolt.Options{Timeout: 1 * time.Second})
	require.NoError(t, err)

	s := BoltDB{DB: db}

	{
		entry := feed.Entry{
			ChannelID: "chan1",
			VideoID:   "vid1",
			Title:     "title1",
			Published: time.Date(2022, time.March, 21, 16, 45, 22, 0, time.UTC),
			File:      "f1",
		}
		e := s.SetProcessed(entry)
		require.NoError(t, e)
	}
	{
		entry := feed.Entry{
			ChannelID: "chan1",
			VideoID:   "vid2",
			Title:     "title2",
			Published: time.Date(2022, time.March, 21, 17, 45, 22, 0, time.UTC),
			File:      "f2",
		}
		e := s.SetProcessed(entry)
		require.NoError(t, e)
	}
	{
		entry := feed.Entry{
			ChannelID: "chan1",
			VideoID:   "vid3",
			Title:     "title3",
			Published: time.Date(2022, time.March, 21, 18, 45, 22, 0, time.UTC),
			File:      "f3",
		}
		e := s.SetProcessed(entry)
		require.NoError(t, e)
	}

	count := s.CountProcessed()
	assert.Equal(t, 3, count)

	found, ts, err := s.CheckProcessed(feed.Entry{ChannelID: "chan1", VideoID: "vid2"})
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, time.Date(2022, time.March, 21, 17, 45, 22, 0, time.UTC), ts)

	found, ts, err = s.CheckProcessed(feed.Entry{ChannelID: "chan1", VideoID: "vid3"})
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, time.Date(2022, time.March, 21, 18, 45, 22, 0, time.UTC), ts)

	found, _, err = s.CheckProcessed(feed.Entry{ChannelID: "chan1", VideoID: "vidXXX"})
	require.NoError(t, err)
	assert.False(t, found)

	lst, err := s.ListProcessed()
	require.NoError(t, err)
	assert.Equal(t, 3, len(lst))
	assert.Equal(t, "dbff863dbc922f727afb93e949704da777739489 / 2022-03-21T16:45:22Z", lst[0])
	assert.Equal(t, "ae9a2ed8bbf505091e35b9d54ccb9dc58e35c205 / 2022-03-21T18:45:22Z", lst[1])
	assert.Equal(t, "a8fd9875c236fb27e26183b6df87f0cecb7a683f / 2022-03-21T17:45:22Z", lst[2])

	err = s.ResetProcessed(feed.Entry{ChannelID: "chan1", VideoID: "vid2"})
	require.NoError(t, err)
	found, _, err = s.CheckProcessed(feed.Entry{ChannelID: "chan1", VideoID: "vid2"})
	require.NoError(t, err)
	assert.False(t, found)

}

func TestBoltDB_Last(t *testing.T) {
	tmpfile := filepath.Join(os.TempDir(), "test.db")
	defer os.Remove(tmpfile)

	db, err := bolt.Open(tmpfile, 0o600, &bolt.Options{Timeout: 1 * time.Second})
	require.NoError(t, err)

	s := BoltDB{DB: db, Channels: []string{"chan1", "chan2"}}
	_, err = s.Last()
	assert.EqualError(t, err, "can't load last entry for chan1: no bucket for chan1")

	{
		entry := feed.Entry{
			ChannelID: "chan1",
			VideoID:   "vid1",
			Title:     "title1",
			Published: time.Date(2022, time.March, 21, 16, 45, 22, 0, time.UTC),
			File:      "f1",
		}
		created, e := s.Save(entry)
		require.NoError(t, e)
		assert.True(t, created)
	}
	{
		entry := feed.Entry{
			ChannelID: "chan1",
			VideoID:   "vid2",
			Title:     "title2",
			Published: time.Date(2022, time.March, 21, 17, 45, 22, 0, time.UTC),
			File:      "f2",
		}
		created, e := s.Save(entry)
		require.NoError(t, e)
		assert.True(t, created)
	}
	{
		entry := feed.Entry{
			ChannelID: "chan2",
			VideoID:   "vid3",
			Title:     "title3",
			Published: time.Date(2022, time.March, 21, 17, 46, 22, 0, time.UTC),
			File:      "f3",
		}
		created, e := s.Save(entry)
		require.NoError(t, e)
		assert.True(t, created)
	}

	res, err := s.Last()
	require.NoError(t, err)
	assert.Equal(t, "vid3", res.VideoID)
}
