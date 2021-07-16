package data

import "log"

var (
	// SilentBytes is the raw data from a 128 Kb/s of lame encoded nothingness
	SilentBytes []byte
)

//go:generate go-bindata -pkg data -nomemcopy ./
func init() {
	var err error
	SilentBytes, err = Asset("silent_1frame.mp3")
	if err != nil {
		log.Fatalf("Could not open silent_1frame.mp3 asset")
	}
}
