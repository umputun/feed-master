package proc

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"time"

	log "github.com/go-pkgz/lgr"
	bolt "go.etcd.io/bbolt"

	"github.com/umputun/feed-master/app/feed"
)

// BoltDB store
type BoltDB struct {
	DB *bolt.DB
}

// Save to bolt, skip if found
func (b BoltDB) Save(fmFeed string, item feed.Item) (bool, error) {
	var created bool

	key, err := func() ([]byte, error) {
		ts, err := time.Parse(time.RFC1123Z, item.PubDate)
		if err != nil {
			return nil, fmt.Errorf("parse pubdate %s: %w", item.PubDate, err)
		}
		h := sha1.New()
		if _, err = h.Write([]byte(item.GUID)); err != nil {
			return nil, fmt.Errorf("hash guid %s: %w", item.GUID, err)
		}
		return fmt.Appendf(nil, "%d-%x", ts.Unix(), h.Sum(nil)), nil
	}()

	if err != nil {
		return created, err
	}

	err = b.DB.Update(func(tx *bolt.Tx) error {
		bucket, e := tx.CreateBucketIfNotExists([]byte(fmFeed))
		if e != nil {
			return fmt.Errorf("create bucket %s: %w", fmFeed, e)
		}
		if bucket.Get(key) != nil {
			return nil
		}

		jdata, jerr := json.Marshal(&item)
		if jerr != nil {
			return fmt.Errorf("marshal item %s: %w", item.GUID, jerr)
		}

		log.Printf("[INFO] save %s - %s - %s - %s", string(key), fmFeed, item.Title, item.GUID)
		e = bucket.Put(key, jdata)
		if e != nil {
			return fmt.Errorf("put item %s: %w", item.GUID, e)
		}

		created = true
		return nil
	})

	if err != nil {
		return created, fmt.Errorf("update db: %w", err)
	}
	return created, nil
}

// Load from bold for given feed, up to max
func (b BoltDB) Load(fmFeed string, maximum int, skipJunk bool) ([]feed.Item, error) {
	var result []feed.Item

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
			if skipJunk && item.Junk {
				continue
			}
			if len(result) >= maximum {
				break
			}
			result = append(result, item)
		}
		return nil
	})
	if err != nil {
		return result, fmt.Errorf("view db: %w", err)
	}
	return result, nil
}

// Remove deletes item matched by GUID from given feed
func (b BoltDB) Remove(fmFeed, guid string) error {
	err := b.DB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(fmFeed))
		if bucket == nil {
			return fmt.Errorf("no bucket for %s", fmFeed)
		}

		// find the item by GUID
		c := bucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			item := feed.Item{}
			if err := json.Unmarshal(v, &item); err != nil {
				log.Printf("[WARN] failed to unmarshal during remove, %v", err)
				continue
			}
			if item.GUID == guid {
				log.Printf("[INFO] remove %s from %s", guid, fmFeed)
				return bucket.Delete(k)
			}
		}

		return fmt.Errorf("item %s not found in %s", guid, fmFeed)
	})
	if err != nil {
		return fmt.Errorf("update db: %w", err)
	}
	return nil
}

func (b BoltDB) removeOld(fmFeed string, keep int) (int, error) {
	var toDelete [][]byte
	deleted := 0

	err := b.DB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(fmFeed))
		if bucket == nil {
			return fmt.Errorf("no bucket for %s", fmFeed)
		}

		recs := 0
		c := bucket.Cursor()
		for k, _ := c.Last(); k != nil; k, _ = c.Prev() {
			recs++
			if recs > keep {
				keyCopy := make([]byte, len(k))
				copy(keyCopy, k)
				toDelete = append(toDelete, keyCopy)
			}
		}

		for _, k := range toDelete {
			if e := bucket.Delete(k); e != nil {
				return fmt.Errorf("delete key: %w", e)
			}
			deleted++
		}

		return nil
	})

	if err != nil {
		return deleted, fmt.Errorf("update db: %w", err)
	}
	return deleted, nil
}
