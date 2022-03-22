package channel

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloader_Get(t *testing.T) {
	lw := bytes.NewBuffer(nil)
	loc := os.TempDir()
	d := NewDownloader("echo {{.ID}} blah {{.FileName}} 12345", lw, loc)
	res, err := d.Get(context.Background(), "id1", "file123")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(loc, "file123.mp3"), res)
	l := lw.String()
	assert.Equal(t, "id1 blah file123 12345\n", l)
	t.Log(l)
}

func TestDownloader_GetFailed(t *testing.T) {
	lw := bytes.NewBuffer(nil)
	loc := os.TempDir()
	d := NewDownloader("no-such-thing {{.ID}} blah {{.FileName}} 12345", lw, loc)
	_, err := d.Get(context.Background(), "id1", "file123")
	require.Error(t, err)
	l := lw.String()
	assert.Contains(t, l, "not found")
	t.Log(l)
}
