// Package feed provided parser and downloader for youtube feeds and entries.
package feed

import (
	"context"
	"encoding/xml"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"time"
)

// Feed represents a YouTube channel feed.
type Feed struct {
	Client          *http.Client
	ChannelBaseURL  string
	PlaylistBaseURL string
}

// Type represents the type of YouTube feed.
type Type string

// enum for the different YouTube feed types.
const (
	FTDefault  = Type("")
	FTChannel  = Type("channel")
	FTPlaylist = Type("playlist")
)

// Get xml/rss feed for channel
// https://www.youtube.com/feeds/videos.xml?channel_id=UCPU28A9z_ka_R5dQfecHJlA
func (c *Feed) Get(ctx context.Context, id string, feedType Type) ([]Entry, error) {

	feedURL, err := c.url(id, feedType)
	if err != nil {
		return nil, fmt.Errorf("failed to get feed url: %w", err)
	}

	req, err := http.NewRequest("GET", feedURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", id, err)
	}
	resp, err := c.Client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get channel %s: %w", id, err)
	}
	defer resp.Body.Close() // nolint
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get %s: %s", id, resp.Status)
	}
	data := struct {
		Entry []Entry `xml:"entry"`
	}{}

	if err := xml.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode %s: %w", id, err)
	}

	sort.Slice(data.Entry, func(i, j int) bool {
		return data.Entry[i].Published.After(data.Entry[j].Published)
	})

	// set channel or playlist id. Need to override this for RSS feed because yt always returns channel id here
	for i := range data.Entry {
		data.Entry[i].ChannelID = id
	}

	return data.Entry, nil
}

func (c *Feed) url(id string, feedType Type) (string, error) {
	switch feedType {
	case FTChannel, FTDefault:
		return c.ChannelBaseURL + id, nil
	case FTPlaylist:
		return c.PlaylistBaseURL + id, nil
	}
	return "", fmt.Errorf("unknown feed type %s", feedType)
}

// Entry represents a YouTube channel entry.
type Entry struct {
	ChannelID string `xml:"http://www.youtube.com/xml/schemas/2015 channelId"`
	VideoID   string `xml:"http://www.youtube.com/xml/schemas/2015 videoId"`
	Title     string `xml:"title"`
	Link      struct {
		Href string `xml:"href,attr"`
	} `xml:"link"`
	Published time.Time `xml:"published"`
	Updated   time.Time `xml:"updated"`

	Media struct {
		Description template.HTML `xml:"description"`
		Thumbnail   struct {
			URL string `xml:"url,attr"`
		} `xml:"thumbnail"`
	} `xml:"http://search.yahoo.com/mrss/ group"`

	Author struct {
		Name string `xml:"name"`
		URI  string `xml:"uri"`
	} `xml:"author"`

	File        string
	Duration    int    // seconds
	DurationFmt string // used for ui only
}

// UID returns the unique identifier of the entry.
func (e *Entry) UID() string {
	return e.ChannelID + "::" + e.VideoID
}

func (e *Entry) String() string {
	tz, _ := time.LoadLocation("Local")

	return fmt.Sprintf("{ChannelID:%s, VideoID:%s, Title:%q, Published:%s, Updated:%s, Author:%s, File:%s, Duration:%ds}",
		e.ChannelID, e.VideoID, e.Title, e.Published.In(tz).Format(time.RFC3339), e.Updated.In(tz).Format(time.RFC3339),
		e.Author.Name, e.File, e.Duration,
	)
}
