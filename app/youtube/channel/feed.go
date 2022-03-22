package channel

import (
	"context"
	"encoding/xml"
	"net/http"

	"github.com/pkg/errors"
)

// Feed represents a YouTube channel feed.
type Feed struct {
	Client  *http.Client
	BaseURL string
}

// Get xml/rss feed for channel
// https://www.youtube.com/feeds/videos.xml?channel_id=UCPU28A9z_ka_R5dQfecHJlA
func (c *Feed) Get(ctx context.Context, chanID string) ([]Entry, error) {
	req, err := http.NewRequest("GET", c.BaseURL+chanID, http.NoBody)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create request for channel %s", chanID)
	}
	resp, err := c.Client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get channel %s", chanID)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("failed to get channel %s: %s", chanID, resp.Status)
	}
	data := struct {
		Entry []Entry `xml:"entry"`
	}{}

	if err := xml.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, errors.Wrapf(err, "failed to decode channel %s", chanID)
	}

	return data.Entry, nil
}
