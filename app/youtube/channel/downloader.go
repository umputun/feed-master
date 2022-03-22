package channel

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

// Downloader executes an external command to download a video and extract its audio.
type Downloader struct {
	ytTemplate  string
	logWriter   io.Writer
	destination string
}

// NewDownloader creates a new Downloader with the given template (full command with placeholders for {{.ID}} and {{.Filename}}.
// Destination is the directory where the audio files will be stored.
func NewDownloader(tmpl string, logWriter io.Writer, destination string) *Downloader {
	return &Downloader{
		ytTemplate:  tmpl,
		logWriter:   logWriter,
		destination: destination,
	}
}

// Get downloads a video from youtube and extracts audio.
// yt-dlp --extract-audio --audio-format=mp3 --audio-quality=0 -f m4a/bestaudio "https://www.youtube.com/watch?v={{.ID}}" --no-progress -o {{.Filename}}.tmp
func (d *Downloader) Get(ctx context.Context, id, fname string) (file string, err error) {

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
	cmd.Stdout = d.logWriter
	cmd.Stderr = d.logWriter
	log.Printf("[DEBUG] executing command: %s", b1.String())
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to execute command: %v", err)
	}
	return filepath.Join(d.destination, fname+".mp3"), nil
}
