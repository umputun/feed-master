package youtube

import (
	"fmt"
	"os"
	"path/filepath"

	log "github.com/go-pkgz/lgr"
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
		return fmt.Errorf("failed to create dir %s: %w", s.Location, err)
	}

	fname := filepath.Join(s.Location, chanID+".xml")
	fh, err := os.Create(fname) //nolint:gosec // tolerable security risk
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", fname, err)
	}
	defer fh.Close() // nolint
	if _, err = fh.WriteString(rss); err != nil {
		return fmt.Errorf("failed to write to file %s: %w", fname, err)
	}
	log.Printf("[INFO] rss feed file saved to %s", fname)
	return nil
}
