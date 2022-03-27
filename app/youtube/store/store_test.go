package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/umputun/feed-master/app/youtube/feed"
	bolt "go.etcd.io/bbolt"
)

func TestStore_SaveAndLoad(t *testing.T) {
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

func TestStore_Channels(t *testing.T) {
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
		}
		created, e := s.Save(entry)
		require.NoError(t, e)
		assert.True(t, created)
	}
	{
		entry := feed.Entry{
			ChannelID: "chan2",
			VideoID:   "vid2",
			Title:     "title2",
			Published: time.Date(2022, time.March, 21, 16, 45, 22, 0, time.UTC),
		}
		created, e := s.Save(entry)
		require.NoError(t, e)
		assert.True(t, created)
	}

	res, err := s.Channels()
	require.NoError(t, err)
	assert.Equal(t, []string{"chan1", "chan2"}, res)
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
