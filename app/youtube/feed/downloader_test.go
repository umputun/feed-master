package feed

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloader_Get(t *testing.T) {
	lw := bytes.NewBuffer(nil)
	loc := os.TempDir()
	fh, err := os.CreateTemp(loc, "downloader_test*.mp3")
	require.NoError(t, err)
	defer os.Remove(fh.Name())

	fname := filepath.Base(fh.Name())

	d := NewDownloader("echo {{.ID}} blah {{.FileName}}.mp3 12345", lw, lw, loc)
	res, err := d.Get(context.Background(), "id1", strings.TrimSuffix(fname, path.Ext(fname)))
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
	assert.Equal(t, fh.Name()+".mp3", res)
}

func TestDownloader_GetFailed(t *testing.T) {
	lw := bytes.NewBuffer(nil)
	loc := os.TempDir()
	fh, err := os.CreateTemp(loc, "downloader_test*.mp3")
	require.NoError(t, err)
	assert.NoError(t, os.Remove(fh.Name()))

	fname := filepath.Base(fh.Name())

	d := NewDownloader("echo {{.ID}} blah {{.FileName}}.mp3 12345", lw, lw, loc)
	res, err := d.Get(context.Background(), "id1", strings.TrimSuffix(fname, path.Ext(fname)))
	require.EqualError(t, err, "skip")
	assert.Equal(t, fh.Name(), res)
}
