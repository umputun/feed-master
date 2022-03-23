package youtube

import (
	"os"
	"path/filepath"

	log "github.com/go-pkgz/lgr"
	"github.com/pkg/errors"
)

// RSSFileStore is a store for RSS feed files
type RSSFileStore struct {
	Location string
	Enabled  bool
}

// Save  RSS feed file to the FS
func (s *RSSFileStore) Save(chanID string, rss string) error {
	if !s.Enabled {
		return nil
	}
	fname := filepath.Join(s.Location, chanID+".xml")
	fh, err := os.Create(fname)
	if err != nil {
		return errors.Wrapf(err, "failed to create file %s", fname)
	}
	defer fh.Close()
	if _, err = fh.WriteString(rss); err != nil {
		return errors.Wrapf(err, "failed to write to file %s", fname)
	}
	log.Printf("[INFO] rss feed file saved to %s", fname)
	return nil
}
