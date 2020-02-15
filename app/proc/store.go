package proc

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/boltdb/bolt"
	log "github.com/go-pkgz/lgr"

	"github.com/umputun/feed-master/app/feed"
)

// BoltDB store
type BoltDB struct {
	DB *bolt.DB
}

// NewBoltDB makes persistent boltdb based store
func NewBoltDB(dbFile string) (*BoltDB, error) {
	log.Printf("[INFO] bolt (persistent) store, %s", dbFile)
	if err := os.MkdirAll(path.Dir(dbFile), 0700); err != nil {
		return nil, err
	}
	result := BoltDB{}
	db, err := bolt.Open(dbFile, 0600, &bolt.Options{Timeout: 1 * time.Second}) // nolint
	if err != nil {
		return nil, err
	}
	result.DB = db
	return &result, err
}

// Save to bolt, skip if found
func (b BoltDB) Save(fmFeed string, item feed.Item) (bool, error) {
	var created bool

	key, err := func() ([]byte, error) {
		ts, err := time.Parse(time.RFC1123Z, item.PubDate)
		if err != nil {
			return nil, err
		}
		h := sha1.New()
		if _, err = h.Write([]byte(item.GUID)); err != nil {
			return nil, err
		}
		return []byte(fmt.Sprintf("%d-%x", ts.Unix(), h.Sum(nil))), nil
	}()

	if err != nil {
		return created, err
	}

	err = b.DB.Update(func(tx *bolt.Tx) error {
		bucket, e := tx.CreateBucketIfNotExists([]byte(fmFeed))
		if e != nil {
			return e
		}
		if bucket.Get(key) != nil {
			return nil
		}

		jdata, jerr := json.Marshal(&item)
		if jerr != nil {
			return jerr
		}

		log.Printf("[INFO] save %s - %s - %s - %s", string(key), fmFeed, item.Title, item.GUID)
		e = bucket.Put(key, jdata)
		if e != nil {
			return e
		}

		created = true
		return e
	})

	return created, err
}

// Load from bold for given feed, up to max
func (b BoltDB) Load(fmFeed string, max int) ([]feed.Item, error) {
	result := []feed.Item{}

	err := b.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(fmFeed))
		if bucket == nil {
			return fmt.Errorf("no bucket for %s", fmFeed)
		}
		c := bucket.Cursor()
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			item := feed.Item{}
			if err := json.Unmarshal(v, &item); err != nil {
				log.Printf("[WARN] failed to unmarshal, %v", err)
				continue
			}
			if len(result) >= max {
				break
			}
			result = append(result, item)
		}
		return nil
	})
	return result, err
}

func (b BoltDB) removeOld(fmFeed string, keep int) (int, error) {
	deleted := 0
	err := b.DB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(fmFeed))
		if bucket == nil {
			return fmt.Errorf("no bucket for %s", fmFeed)
		}
		recs := 0
		c := bucket.Cursor()
		var err error
		for k, _ := c.Last(); k != nil; k, _ = c.Prev() {
			recs++
			if recs > keep {
				if e := bucket.Delete(k); e != nil {
					err = e
				}
				deleted++
			}
		}
		return err
	})
	return deleted, err
}

// Buckets returns list of buckets
func (b BoltDB) Buckets() (result []string, err error) {
	err = b.DB.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, _ *bolt.Bucket) error { // nolint
			result = append(result, string(name))
			return nil
		})
	})
	return result, err
}
