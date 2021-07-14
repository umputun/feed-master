package feed

import (
	"html/template"
	"io"
	"net/http"
	"path"
	"time"
)

// Item for rss
type Item struct {
	// Required
	Title       string        `xml:"title"`
	Link        string        `xml:"link"`
	Description template.HTML `xml:"description"`
	// Optional
	Content   template.HTML `xml:"encoded"`
	PubDate   string        `xml:"pubDate"`
	Comments  string        `xml:"comments"`
	Enclosure Enclosure     `xml:"enclosure"`
	GUID      string        `xml:"guid"`

	// Internal
	DT   time.Time `xml:"-"`
	Junk bool      `xml:"-"`
}

// DownloadAudio return httpBody for Item's Enclosure.URL
func (item Item) DownloadAudio(timeout time.Duration) (io.ReadCloser, error) {
	clientHTTP := &http.Client{Timeout: timeout}

	resp, err := clientHTTP.Get(item.Enclosure.URL)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

// GetFilename returns the filename for Item's Enclosure.URL
func (item Item) GetFilename() string {
	_, filename := path.Split(item.Enclosure.URL)
	return filename
}
