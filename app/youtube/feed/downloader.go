package feed

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	log "github.com/go-pkgz/lgr"

	"github.com/pkg/errors"
)

// ErrSkip is returned when the file is not downloaded
var ErrSkip = errors.New("skip")

// Downloader executes an external command to download a video and extract its audio.
type Downloader struct {
	ytTemplate   string
	logOutWriter io.Writer
	logErrWriter io.Writer
	destination  string
}

// NewDownloader creates a new Downloader with the given template (full command with placeholders for {{.ID}} and {{.Filename}}.
// Destination is the directory where the audio files will be stored.
func NewDownloader(tmpl string, logOutWriter, logErrWriter io.Writer, destination string) *Downloader {
	return &Downloader{
		ytTemplate:   tmpl,
		logOutWriter: logOutWriter,
		logErrWriter: logErrWriter,
		destination:  destination,
	}
}

// Get downloads a video from youtube and extracts audio.
// yt-dlp --extract-audio --audio-format=mp3 --audio-quality=0 -f m4a/bestaudio "https://www.youtube.com/watch?v={{.ID}}" --no-progress -o {{.Filename}}
func (d *Downloader) Get(ctx context.Context, id, fname string) (file string, err error) {

	if err := os.MkdirAll(d.destination, 0o750); err != nil {
		return "", errors.Wrapf(err, "failed to create directory %s", d.destination)
	}

	tmplParams := struct {
		ID       string
		FileName string
	}{
		ID:       id,
		FileName: fname,
	}
	b1 := bytes.Buffer{}
	if err := template.Must(template.New("youtube-dl").Parse(d.ytTemplate)).Execute(&b1, tmplParams); err != nil { // nolint
		return "", fmt.Errorf("failed to parse template: %v", err)
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", b1.String()) // nolint
	cmd.Stdin = os.Stdin
	cmd.Stdout = d.logOutWriter
	cmd.Stderr = d.logErrWriter
	cmd.Dir = d.destination
	log.Printf("[DEBUG] executing command: %s", b1.String())
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to execute command: %v", err)
	}

	file = filepath.Join(d.destination, fname+".mp3")
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return file, ErrSkip
	}
	return file, nil
}
