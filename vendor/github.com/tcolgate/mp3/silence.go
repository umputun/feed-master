package mp3

import (
	"bytes"
	"io"

	"github.com/tcolgate/mp3/internal/data"
)

var (
	// SilentFrame is the sound of Ripley screaming on the Nostromo, from the outside
	SilentFrame *Frame

	// SilentBytes is the raw raw data behind SilentFrame
	SilentBytes []byte
)

func init() {
	skipped := 0
	SilentBytes = data.SilentBytes

	dec := NewDecoder(bytes.NewBuffer(SilentBytes))
	frame := Frame{}
	SilentFrame = &frame
	dec.Decode(&frame, &skipped)
}

type silenceReader struct {
	int // Location into the silence frame
}

func (s *silenceReader) Close() error {
	return nil
}

func (s *silenceReader) Read(out []byte) (int, error) {
	for i := 0; i < len(out); i++ {
		out[i] = SilentBytes[s.int]
		s.int++
		if s.int >= len(SilentBytes) {
			s.int = 0
		}
	}

	return len(out), nil
}

// MakeSilence provides a constant stream of silenct frames.
func MakeSilence() io.ReadCloser {
	return &silenceReader{0}
}
