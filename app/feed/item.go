package feed

import (
	"fmt"
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
	Enclosure   Enclosure     `xml:"enclosure"`
	GUID        string        `xml:"guid"`
	// Optional
	Content  template.HTML `xml:"encoded,omitempty"`
	PubDate  string        `xml:"pubDate,omitempty"`
	Comments string        `xml:"comments,omitempty"`
	Author   string        `xml:"author,omitempty"`
	Duration string        `xml:"duration,omitempty"`
	// Internal
	DT          time.Time `xml:"-"`
	Junk        bool      `xml:"-"`
	DurationFmt string    `xml:"-"` // used for ui only in
}

// DownloadAudio return httpBody for Item's Enclosure.URL
func (item Item) DownloadAudio(timeout time.Duration) (io.ReadCloser, error) {
	clientHTTP := &http.Client{Timeout: timeout}

	resp, err := clientHTTP.Get(item.Enclosure.URL)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("incorrect status code %s for %s", resp.Status, item.Enclosure.URL)
	}

	return resp.Body, nil
}

// GetFilename returns the filename for Item's Enclosure.URL
func (item Item) GetFilename() string {
	_, filename := path.Split(item.Enclosure.URL)
	return filename
}
