package duration

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestService_File(t *testing.T) {
	svc := Service{}
	{
		res := svc.File("testdata/audio.mp3")
		assert.Equal(t, 47, res)
	}

	{
		res := svc.File("testdata/no-file.mp3")
		assert.Equal(t, 0, res)
	}
}

func TestService_Reader(t *testing.T) {
	// taken from https://github.com/mathiasbynens/small/blob/master/mp3.mp3
	smallMP3File := []byte{54, 53, 53, 48, 55, 54, 51, 52, 48, 48, 51, 49, 56, 52, 51, 50, 48, 55, 54, 49, 54, 55, 49, 55, 49, 55, 55, 49, 53, 49, 49, 56, 51, 51, 49, 52, 51, 56, 50, 49, 50, 56, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48}
	reader := bytes.NewReader(smallMP3File)

	svc := Service{}
	assert.Zero(t, svc.reader(reader))
}
