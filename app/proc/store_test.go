package proc

import (
	"os"
	"strconv"
	"testing"
	"time"

	bolt "go.etcd.io/bbolt"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/feed-master/app/feed"
)

const pubDate = "Mon, 02 Jan 2006 15:04:05 -0700"

func TestSaveIfInvalidPubDate(t *testing.T) {
	tmpfile, _ := os.CreateTemp("", "")
	defer os.Remove(tmpfile.Name())

	db, err := bolt.Open(tmpfile.Name(), 0o600, &bolt.Options{Timeout: 1 * time.Second}) // nolint
	require.NoError(t, err)
	bdb := &BoltDB{DB: db}

	item := feed.Item{
		PubDate: "100500",
	}
	created, err := bdb.Save("radio-t", item)
	assert.False(t, created)
	assert.EqualError(t, err, "parsing time \"100500\" as \"Mon, 02 Jan 2006 15:04:05 -0700\": cannot parse \"100500\" as \"Mon\"")
}

func TestSave(t *testing.T) {
	tmpfile, _ := os.CreateTemp("", "")
	defer os.Remove(tmpfile.Name())
	db, err := bolt.Open(tmpfile.Name(), 0o600, &bolt.Options{Timeout: 1 * time.Second}) // nolint
	require.NoError(t, err)
	bdb := &BoltDB{DB: db}

	item := feed.Item{
		PubDate: pubDate,
	}

	created, err := bdb.Save("radio-t", item)

	assert.True(t, created)
	assert.NoError(t, err)
}

func TestSaveIfItemIsExists(t *testing.T) {
	tmpfile, _ := os.CreateTemp("", "")
	defer os.Remove(tmpfile.Name())
	db, err := bolt.Open(tmpfile.Name(), 0o600, &bolt.Options{Timeout: 1 * time.Second}) // nolint
	require.NoError(t, err)
	bdb := &BoltDB{DB: db}

	item := feed.Item{
		PubDate: pubDate,
	}
	_, err = bdb.Save("radio-t", item)
	require.NoError(t, err)

	created, err := bdb.Save("radio-t", item)

	assert.False(t, created)
	assert.NoError(t, err)
}

func TestLoadIfNotBucket(t *testing.T) {
	tmpfile, _ := os.CreateTemp("", "")
	defer os.Remove(tmpfile.Name())
	db, err := bolt.Open(tmpfile.Name(), 0o600, &bolt.Options{Timeout: 1 * time.Second}) // nolint
	require.NoError(t, err)
	bdb := &BoltDB{DB: db}

	feedItems, err := bdb.Load("100500", 5, false)

	assert.Equal(t, len(feedItems), 0)
	assert.EqualError(t, err, "no bucket for 100500")
}

func TestLoad(t *testing.T) {
	tmpfile, _ := os.CreateTemp("", "")
	defer os.Remove(tmpfile.Name())
	db, err := bolt.Open(tmpfile.Name(), 0o600, &bolt.Options{Timeout: 1 * time.Second}) // nolint
	require.NoError(t, err)
	bdb := &BoltDB{DB: db}

	_, err = bdb.Save("radio-t", feed.Item{PubDate: pubDate})
	require.NoError(t, err)

	items, err := bdb.Load("radio-t", 5, false)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(items))
	assert.Equal(t, items[0].PubDate, pubDate)
}

func TestLoadChackMax(t *testing.T) {
	tmpfile, _ := os.CreateTemp("", "")
	defer os.Remove(tmpfile.Name())
	db, err := bolt.Open(tmpfile.Name(), 0o600, &bolt.Options{Timeout: 1 * time.Second}) // nolint
	require.NoError(t, err)
	bdb := &BoltDB{DB: db}

	_, err = bdb.Save("radio-t", feed.Item{PubDate: pubDate, GUID: "1"})
	require.NoError(t, err)

	_, err = bdb.Save("radio-t", feed.Item{PubDate: pubDate, GUID: "2"})
	require.NoError(t, err)

	cases := []struct {
		max   int
		count int
	}{
		{0, 0},
		{1, 1},
		{2, 2},
		{5, 2},
	}

	for i, tc := range cases {
		i := i
		tc := tc
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			items, err := bdb.Load("radio-t", tc.max, false)

			assert.NoError(t, err)
			assert.Equal(t, tc.count, len(items))
		})
	}
}

func TestRemoveOldIfNotExistsBucket(t *testing.T) {
	tmpfile, _ := os.CreateTemp("", "")
	defer os.Remove(tmpfile.Name())
	db, err := bolt.Open(tmpfile.Name(), 0o600, &bolt.Options{Timeout: 1 * time.Second}) // nolint
	require.NoError(t, err)
	bdb := &BoltDB{DB: db}

	count, err := bdb.removeOld("radio-t", 5)

	assert.EqualError(t, err, "no bucket for radio-t")
	assert.Equal(t, 0, count)
}

func TestRemoveOld(t *testing.T) {
	cases := []struct {
		keep        int
		countDelete int
	}{
		{0, 2},
		{1, 1},
		{2, 0},
	}

	for i, tc := range cases {
		i := i
		tc := tc
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			tmpfile, _ := os.CreateTemp("", "")
			defer os.Remove(tmpfile.Name())
			db, err := bolt.Open(tmpfile.Name(), 0o600, &bolt.Options{Timeout: 1 * time.Second}) // nolint
			require.NoError(t, err)
			bdb := &BoltDB{DB: db}
			_, err = bdb.Save("radio-t", feed.Item{PubDate: pubDate, GUID: "1"})
			require.NoError(t, err)

			_, err = bdb.Save("radio-t", feed.Item{PubDate: pubDate, GUID: "2"})
			require.NoError(t, err)

			count, err := bdb.removeOld("radio-t", tc.keep)

			assert.NoError(t, err)
			assert.Equal(t, tc.countDelete, count)
		})
	}
}

func TestRemoveOldKeepsNewestItems(t *testing.T) {
	tmpfile, _ := os.CreateTemp("", "")
	defer os.Remove(tmpfile.Name())
	db, err := bolt.Open(tmpfile.Name(), 0o600, &bolt.Options{Timeout: 1 * time.Second}) // nolint
	require.NoError(t, err)
	defer db.Close()
	bdb := &BoltDB{DB: db}

	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 100; i++ {
		itemTime := baseTime.Add(time.Duration(i) * time.Hour)
		item := feed.Item{PubDate: itemTime.Format(time.RFC1123Z), GUID: "item-" + strconv.Itoa(i), Title: "Item " + strconv.Itoa(i)}
		_, err = bdb.Save("test-feed", item)
		require.NoError(t, err)
	}

	count, err := bdb.removeOld("test-feed", 50)
	require.NoError(t, err)
	assert.Equal(t, 50, count)

	items, err := bdb.Load("test-feed", 100, false)
	require.NoError(t, err)
	require.Equal(t, 50, len(items))

	for i, item := range items {
		expectedGUID := "item-" + strconv.Itoa(99-i)
		assert.Equal(t, expectedGUID, item.GUID, "item at position %d should be %s but got %s", i, expectedGUID, item.GUID)
	}
}

func TestRemoveOldRepeatedCycles(t *testing.T) {
	tmpfile, _ := os.CreateTemp("", "")
	defer os.Remove(tmpfile.Name())
	db, err := bolt.Open(tmpfile.Name(), 0o600, &bolt.Options{Timeout: 1 * time.Second}) // nolint
	require.NoError(t, err)
	defer db.Close()
	bdb := &BoltDB{DB: db}

	baseTime := time.Date(2024, 12, 29, 0, 0, 0, 0, time.UTC)
	item1 := feed.Item{PubDate: baseTime.Format(time.RFC1123Z), GUID: "item-old-1", Title: "Old Item 1"}
	item2 := feed.Item{PubDate: baseTime.Add(1 * time.Hour).Format(time.RFC1123Z), GUID: "item-old-2", Title: "Old Item 2"}

	for cycle := 0; cycle < 5; cycle++ {
		created1, err := bdb.Save("test-feed", item1)
		require.NoError(t, err)
		created2, err := bdb.Save("test-feed", item2)
		require.NoError(t, err)

		if cycle == 0 {
			assert.True(t, created1, "cycle %d: item1 should be created on first cycle", cycle)
			assert.True(t, created2, "cycle %d: item2 should be created on first cycle", cycle)
		} else {
			assert.False(t, created1, "cycle %d: item1 should NOT be recreated, but created=%v", cycle, created1)
			assert.False(t, created2, "cycle %d: item2 should NOT be recreated, but created=%v", cycle, created2)
		}

		_, err = bdb.removeOld("test-feed", 5)
		require.NoError(t, err)
	}
}
