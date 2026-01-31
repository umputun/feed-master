// Package duration provides a duration of audio from file or reader
package duration

import (
	"errors"
	"io"
	"os"

	log "github.com/go-pkgz/lgr"
	"github.com/tcolgate/mp3"
)

// Service provides duration of audio from file or reader
type Service struct{}

// File scans MP3 file from provided file and returns its duration in seconds, ignoring possible errors
func (s *Service) File(fname string) int {
	fh, err := os.Open(fname) //nolint:gosec // file path is internal, not from user input
	if err != nil {
		log.Printf("[WARN] can't get duration, failed to open file %s: %v", fname, err)
		return 0
	}
	defer fh.Close()
	return s.reader(fh)
}

// reader scans MP3 from provided file and returns its duration in seconds, ignoring possible errors
func (s *Service) reader(r io.Reader) int {
	d := mp3.NewDecoder(r)

	var f mp3.Frame
	var skipped int
	var duration float64
	var err error

	for err == nil {
		if err = d.Decode(&f, &skipped); err != nil && !errors.Is(err, io.EOF) {
			log.Printf("[WARN] can't get duration for provided stream: %v", err)
			return 0
		}
		duration += f.Duration().Seconds()
	}
	return int(duration)
}
