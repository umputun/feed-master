// Package store provides a store for the youtube service metadata
package store

import (
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/hashicorp/go-multierror"
	bolt "go.etcd.io/bbolt"

	"github.com/umputun/feed-master/app/youtube/feed"
)

var processedBkt = []byte("processed")

// BoltDB store for metadata related to downloaded YouTube audio.
type BoltDB struct {
	*bolt.DB
	Channels []string // the list of configured channels ids
}

// Save to bolt, skip if found
func (s *BoltDB) Save(entry feed.Entry) (bool, error) {
	var created bool

	key, keyErr := s.key(entry)
	if keyErr != nil {
		return created, fmt.Errorf("failed to generate key for %s: %w", entry.VideoID, keyErr)
	}

	err := s.Update(func(tx *bolt.Tx) error {
		bucket, e := tx.CreateBucketIfNotExists([]byte(entry.ChannelID))
		if e != nil {
			return fmt.Errorf("create bucket %s: %w", entry.ChannelID, e)
		}
		if bucket.Get(key) != nil {
			return nil
		}

		jdata, jerr := json.Marshal(&entry)
		if jerr != nil {
			return fmt.Errorf("marshal entry %s: %w", entry.VideoID, jerr)
		}

		log.Printf("[INFO] save %s - %s", string(key), entry.String())

		e = bucket.Put(key, jdata)
		if e != nil {
			return fmt.Errorf("save entry %s: %w", entry.VideoID, e)
		}

		created = true
		return e
	})

	return created, err
}

// Exist checks if entry exists
func (s *BoltDB) Exist(entry feed.Entry) (bool, error) {
	var found bool

	key, keyErr := s.key(entry)
	if keyErr != nil {
		return false, fmt.Errorf("failed to generate key for %s: %w", entry.VideoID, keyErr)
	}

	err := s.View(func(tx *bolt.Tx) error {
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
func (s *BoltDB) Load(channelID string, maximum int) ([]feed.Entry, error) {
	var result []feed.Entry

	err := s.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(channelID))
		if bucket == nil {
			return fmt.Errorf("no bucket for %s", channelID)
		}
		c := bucket.Cursor()
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			var item feed.Entry
			if err := json.Unmarshal(v, &item); err != nil {
				log.Printf("[WARN] failed to unmarshal %s, %q: %v", channelID, string(v), err)
				continue
			}
			if len(result) >= maximum {
				break
			}
			result = append(result, item)
		}
		return nil
	})
	return result, err
}

// Last returns last (newest) entry across all channels
func (s *BoltDB) Last() (feed.Entry, error) {
	entries := []feed.Entry{}
	for _, channel := range s.Channels {
		last, err := s.Load(channel, 1)
		if err != nil {
			return feed.Entry{}, fmt.Errorf("can't load last entry for %s: %w", channel, err)
		}
		if len(last) > 0 {
			entries = append(entries, last[0])
		}
	}
	if len(entries) == 0 {
		return feed.Entry{}, errors.New("no entries")
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Published.After(entries[j].Published)
	})
	return entries[0], nil
}

// RemoveOld removes old entries from bolt and returns the list of removed entry.File
// the caller should delete the files
// important: this method returns the list of removed keys even if there was an error
func (s *BoltDB) RemoveOld(channelID string, keep int) ([]string, error) {
	deleted := 0
	var res []string

	err := s.Update(func(tx *bolt.Tx) (e error) {
		errs := new(multierror.Error)
		bucket := tx.Bucket([]byte(channelID))
		if bucket == nil {
			return fmt.Errorf("no bucket for %s", channelID)
		}
		recs := 0
		c := bucket.Cursor()
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			recs++
			if recs > keep {
				var item feed.Entry
				if err := json.Unmarshal(v, &item); err != nil {
					log.Printf("[WARN] failed to unmarshal, %v", err)
					continue
				}
				if err := bucket.Delete(k); err != nil {
					errs = multierror.Append(errs, fmt.Errorf("failed to delete %s (%s): %w", string(k), item.File, err))
					continue
				}
				res = append(res, item.File)
				deleted++
			}
		}
		return errs.ErrorOrNil()
	})

	return res, err
}

// Remove entry matched by vidoID and channelID
func (s *BoltDB) Remove(entry feed.Entry) error {

	err := s.Update(func(tx *bolt.Tx) (e error) {
		bucket := tx.Bucket([]byte(entry.ChannelID))
		if bucket == nil {
			return fmt.Errorf("no bucket for %s", entry.ChannelID)
		}
		c := bucket.Cursor()
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			var item feed.Entry
			if err := json.Unmarshal(v, &item); err != nil {
				log.Printf("[WARN] failed to unmarshal, %v", err)
				continue
			}
			if item.VideoID == entry.VideoID {
				if err := bucket.Delete(k); err != nil {
					return fmt.Errorf("failed to delete %s (%s): %w", string(k), item.VideoID, err)
				}
				log.Printf("[INFO] delete %s - %s", string(k), item.String())
			}
			return nil
		}
		return nil
	})

	return err
}

// SetProcessed sets processed status with ts for a given channel+video
func (s *BoltDB) SetProcessed(entry feed.Entry) error {

	key, keyErr := s.procKey(entry)
	if keyErr != nil {
		return fmt.Errorf("failed to generate key for %s: %w", entry.VideoID, keyErr)
	}

	err := s.Update(func(tx *bolt.Tx) error {
		bucket, e := tx.CreateBucketIfNotExists(processedBkt)
		if e != nil {
			return fmt.Errorf("create bucket %s: %w", processedBkt, e)
		}
		if bucket.Get(key) != nil {
			return nil
		}

		log.Printf("[INFO] set processed %s - %s", string(key), entry.String())

		e = bucket.Put(key, []byte(entry.Published.Format(time.RFC3339)))
		if e != nil {
			return fmt.Errorf("save processed %s: %w", entry.VideoID, e)
		}
		return e
	})

	return err
}

// ResetProcessed resets processed status for a given channel+video
func (s *BoltDB) ResetProcessed(entry feed.Entry) error {

	key, keyErr := s.procKey(entry)
	if keyErr != nil {
		return fmt.Errorf("failed to generate key for %s: %w", entry.VideoID, keyErr)
	}

	err := s.Update(func(tx *bolt.Tx) error {
		bucket, e := tx.CreateBucketIfNotExists(processedBkt)
		if e != nil {
			return fmt.Errorf("create bucket %s: %w", processedBkt, e)
		}
		if bucket.Get(key) == nil {
			return nil
		}

		log.Printf("[INFO] reset processed %s - %s", string(key), entry.String())

		e = bucket.Delete(key)
		if e != nil {
			return fmt.Errorf("reset processed %s: %w", entry.VideoID, e)
		}
		return e
	})

	return err
}

// CheckProcessed get processed status and returns timestamp for a given channel+video
// returns found=true if was set before and also the timestamp from stored entry.Published
func (s *BoltDB) CheckProcessed(entry feed.Entry) (found bool, ts time.Time, err error) {

	key, keyErr := s.procKey(entry)
	if keyErr != nil {
		return false, time.Time{}, fmt.Errorf("failed to generate key for %s: %w", entry.VideoID, keyErr)
	}

	err = s.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(processedBkt)
		if bucket == nil {
			return nil
		}

		res := bucket.Get(key)
		if res == nil {
			found = false
			return nil
		}
		found = true
		var tsErr error
		ts, tsErr = time.Parse(time.RFC3339, string(res))
		return tsErr
	})

	return found, ts, err
}

// CountProcessed returns the number of processed entries stored in processedBkt
func (s *BoltDB) CountProcessed() (count int) {

	_ = s.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(processedBkt)
		if bucket == nil {
			return nil
		}

		count = bucket.Stats().KeyN
		return nil
	})
	return count
}

// ListProcessed returns processed entries stored in processedBkt
func (s *BoltDB) ListProcessed() (res []string, err error) {

	err = s.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(processedBkt)
		if bucket == nil {
			return nil
		}
		c := bucket.Cursor()
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			res = append(res, string(k)+" / "+string(v))
		}
		return nil
	})
	return res, err
}

func (s *BoltDB) key(entry feed.Entry) ([]byte, error) {
	h := sha1.New()
	if _, err := h.Write([]byte(entry.VideoID)); err != nil {
		return nil, err
	}
	return []byte(fmt.Sprintf("%d-%x", entry.Published.Unix(), h.Sum(nil))), nil
}

func (s *BoltDB) procKey(entry feed.Entry) ([]byte, error) {
	h := sha1.New()
	if _, err := h.Write([]byte(entry.ChannelID + "::" + entry.VideoID)); err != nil {
		return nil, err
	}
	return []byte(fmt.Sprintf("%x", h.Sum(nil))), nil
}
