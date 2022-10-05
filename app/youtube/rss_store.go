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

// Save RSS feed file to the FS
func (s *RSSFileStore) Save(chanID, rss string) error {
	if !s.Enabled {
		return nil
	}
	if err := os.MkdirAll(s.Location, 0o750); err != nil {
		return errors.Wrapf(err, "failed to create dir %s", s.Location)
	}

	fname := filepath.Join(s.Location, chanID+".xml")
	fh, err := os.Create(fname) //nolint:gosec // tolerable security risk
	if err != nil {
		return errors.Wrapf(err, "failed to create file %s", fname)
	}
	defer fh.Close() // nolint
	if _, err = fh.WriteString(rss); err != nil {
		return errors.Wrapf(err, "failed to write to file %s", fname)
	}
	log.Printf("[INFO] rss feed file saved to %s", fname)
	return nil
}
