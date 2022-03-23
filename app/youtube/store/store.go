// Package store provides a store for the youtube service metadata
package store

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"

	log "github.com/go-pkgz/lgr"
	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"

	"github.com/umputun/feed-master/app/youtube/channel"
)

// BoltDB store for metadata related to downloaded YouTube audio.
type BoltDB struct {
	*bolt.DB
}

// Save to bolt, skip if found
func (s *BoltDB) Save(entry channel.Entry) (bool, error) {
	var created bool

	key, keyErr := func() ([]byte, error) {
		h := sha1.New()
		if _, err := h.Write([]byte(entry.VideoID)); err != nil {
			return nil, err
		}
		return []byte(fmt.Sprintf("%d-%x", entry.Published.Unix(), h.Sum(nil))), nil
	}()

	if keyErr != nil {
		return created, errors.Wrapf(keyErr, "failed to generate key for %s", entry.VideoID)
	}

	err := s.DB.Update(func(tx *bolt.Tx) error {
		bucket, e := tx.CreateBucketIfNotExists([]byte(entry.ChannelID))
		if e != nil {
			return errors.Wrapf(e, "create bucket %s", entry.ChannelID)
		}
		if bucket.Get(key) != nil {
			return nil
		}

		jdata, jerr := json.Marshal(&entry)
		if jerr != nil {
			return errors.Wrapf(jerr, "marshal entry %s", entry.VideoID)
		}

		log.Printf("[INFO] save %s - {ChannelID:%s, VideoID:%s, Title:%s, File:%s, Author:%s, Published:%s}",
			string(key), entry.ChannelID, entry.VideoID, entry.Title, entry.File, entry.Author.Name, entry.Published)

		e = bucket.Put(key, jdata)
		if e != nil {
			return errors.Wrapf(e, "save entry %s", entry.VideoID)
		}

		created = true
		return e
	})

	return created, err
}

// Exist checks if entry exists
func (s *BoltDB) Exist(entry channel.Entry) (bool, error) {
	var found bool

	key, keyErr := func() ([]byte, error) {
		h := sha1.New()
		if _, err := h.Write([]byte(entry.VideoID)); err != nil {
			return nil, err
		}
		return []byte(fmt.Sprintf("%d-%x", entry.Published.Unix(), h.Sum(nil))), nil
	}()

	if keyErr != nil {
		return found, errors.Wrapf(keyErr, "failed to generate key for %s", entry.VideoID)
	}

	err := s.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(entry.ChannelID))
		if bucket == nil {
			return nil
		}

		if bucket.Get(key) != nil {
			found = true
		}

		return nil
	})

	return found, err
}

// Load entries from bolt for a given channel, up to max in reverse order (from newest to oldest)
func (s *BoltDB) Load(channelID string, max int) ([]channel.Entry, error) {
	var result []channel.Entry

	err := s.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(channelID))
		if bucket == nil {
			return fmt.Errorf("no bucket for %s", channelID)
		}
		c := bucket.Cursor()
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			var item channel.Entry
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

// Channels returns list of channels (buckets)
func (s *BoltDB) Channels() (result []string, err error) {
	err = s.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, _ *bolt.Bucket) error { // nolint
			result = append(result, string(name))
			return nil
		})
	})
	return result, err
}

// RemoveOld removes old entries from bolt and returns the list of removed entry.File
// the caller should delete the files
func (s *BoltDB) RemoveOld(channelID string, keep int) ([]string, error) {
	deleted := 0
	var res []string

	err := s.DB.Update(func(tx *bolt.Tx) (e error) {
		bucket := tx.Bucket([]byte(channelID))
		if bucket == nil {
			return fmt.Errorf("no bucket for %s", channelID)
		}
		recs := 0
		c := bucket.Cursor()
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			recs++
			if recs > keep {
				var item channel.Entry
				if err := json.Unmarshal(v, &item); err != nil {
					log.Printf("[WARN] failed to unmarshal, %v", err)
					continue
				}
				res = append(res, item.File)

				e = bucket.Delete(k)
				deleted++
			}
		}
		return e
	})
	return res, err
}
