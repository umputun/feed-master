package feed

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"path"
	"time"

	"github.com/go-pkgz/repeater"
	"github.com/pkg/errors"
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
func (item Item) DownloadAudio(timeout time.Duration) (res io.ReadCloser, err error) {
	clientHTTP := &http.Client{Timeout: timeout}

	rp := repeater.NewDefault(10, time.Second)
	err = rp.Do(context.Background(), func() error {
		resp, e := clientHTTP.Get(item.Enclosure.URL)
		if e != nil {
			return errors.Wrapf(e, "can't download %s", item.Enclosure.URL)
		}
		if resp.StatusCode != http.StatusOK {
			_ = resp.Body.Close()
			return fmt.Errorf("incorrect status code %s for %s", resp.Status, item.Enclosure.URL)
		}
		res = resp.Body
		return nil
	})

	return res, err
}

// GetFilename returns the filename for Item's Enclosure.URL
func (item Item) GetFilename() string {
	_, filename := path.Split(item.Enclosure.URL)
	return filename
}
