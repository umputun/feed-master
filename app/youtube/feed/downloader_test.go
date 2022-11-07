package feed

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloader_Get(t *testing.T) {
	lw := bytes.NewBuffer(nil)
	loc := os.TempDir()
	fh, err := os.CreateTemp(loc, "downloader_test")
	require.NoError(t, err)
	defer os.Remove(fh.Name())

	fname := filepath.Base(fh.Name())

	d := NewDownloader("echo {{.ID}} blah {{.FileName}} 12345", lw, lw, loc)
	res, err := d.Get(context.Background(), "id1", fname)
	require.NoError(t, err)
	assert.Equal(t, fh.Name(), res)
	l := lw.String()
	assert.Equal(t, fmt.Sprintf("id1 blah %s 12345\n", fname), l)
	t.Log(l)
}

func TestDownloader_GetSkip(t *testing.T) {
	lw := bytes.NewBuffer(nil)
	loc := os.TempDir()
	fh, err := os.CreateTemp(loc, "downloader_test")
	require.NoError(t, err)
	assert.NoError(t, os.Remove(fh.Name()))

	fname := filepath.Base(fh.Name())
	d := NewDownloader("echo {{.ID}} blah {{.FileName}} 12345", lw, lw, loc)
	res, err := d.Get(context.Background(), "id1", fname)
	require.EqualError(t, err, "skip")
	assert.Equal(t, fh.Name(), res)
}

func TestDownloader_GetFailed(t *testing.T) {
	lw := bytes.NewBuffer(nil)
	loc := os.TempDir()
	d := NewDownloader("no-such-thing {{.ID}} blah {{.FileName}} 12345", lw, lw, loc)
	_, err := d.Get(context.Background(), "id1", "file123")
	require.Error(t, err)
	l := lw.String()
	assert.Contains(t, l, "not found")
	t.Log(l)
}
