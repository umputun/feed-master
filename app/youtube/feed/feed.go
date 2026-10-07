// Package feed provided parser and downloader for youtube feeds and entries.
package feed

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	log "github.com/go-pkgz/lgr"
)

// Feed represents a YouTube channel feed.
type Feed struct {
	Client          *http.Client
	ChannelBaseURL  string
	PlaylistBaseURL string
	// APIKey is an optional YouTube Data API key, used to read the same videos through the API
	// when the RSS feed request fails
	APIKey     string
	APIBaseURL string // defaults to youtubeAPIBaseURL, overridden in tests
}

const youtubeAPIBaseURL = "https://www.googleapis.com/youtube/v3"

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

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", id, err)
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel %s: %w", id, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if c.APIKey != "" {
			// youtube's feeds/videos.xml can return 404 for every channel; fall back to the data api
			log.Printf("[WARN] feed for %s returned %s, reading it with the youtube api", id, resp.Status)
			return c.getFromAPI(ctx, id, feedType)
		}
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

// getFromAPI reads the latest videos of a channel (its uploads playlist) or playlist with the
// youtube data api playlistItems.list call (1 quota unit) and returns them as feed entries
func (c *Feed) getFromAPI(ctx context.Context, id string, feedType Type) ([]Entry, error) {
	playlistID := id
	if feedType != FTPlaylist {
		// a channel's uploads playlist id is its channel id with UC replaced by UU
		playlistID = "UU" + strings.TrimPrefix(id, "UC")
	}

	baseURL := c.APIBaseURL
	if baseURL == "" {
		baseURL = youtubeAPIBaseURL
	}
	query := url.Values{}
	query.Set("part", "snippet,contentDetails")
	query.Set("maxResults", "50")
	query.Set("playlistId", playlistID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/playlistItems?"+query.Encode(), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create api request for %s: %w", id, err)
	}
	req.Header.Set("X-Goog-Api-Key", c.APIKey) // header, so the key never shows up in logged urls
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get %s from youtube api: %w", id, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get %s from youtube api: %s", id, resp.Status)
	}

	data := struct {
		Items []struct {
			Snippet struct {
				Title                  string `json:"title"`
				Description            string `json:"description"`
				VideoOwnerChannelTitle string `json:"videoOwnerChannelTitle"`
				VideoOwnerChannelID    string `json:"videoOwnerChannelId"`
				Thumbnails             map[string]struct {
					URL string `json:"url"`
				} `json:"thumbnails"`
			} `json:"snippet"`
			ContentDetails struct {
				VideoID          string    `json:"videoId"`
				VideoPublishedAt time.Time `json:"videoPublishedAt"`
			} `json:"contentDetails"`
		} `json:"items"`
	}{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode youtube api response for %s: %w", id, err)
	}

	entries := make([]Entry, 0, len(data.Items))
	for _, item := range data.Items {
		// private and deleted videos stay in playlists without a publish date
		if item.ContentDetails.VideoPublishedAt.IsZero() {
			continue
		}
		e := Entry{
			ChannelID: id,
			VideoID:   item.ContentDetails.VideoID,
			Title:     item.Snippet.Title,
			Published: item.ContentDetails.VideoPublishedAt,
			Updated:   item.ContentDetails.VideoPublishedAt,
		}
		e.Link.Href = "https://www.youtube.com/watch?v=" + item.ContentDetails.VideoID
		e.Media.Description = template.HTML(template.HTMLEscapeString(item.Snippet.Description)) //nolint:gosec // escaped above
		e.Media.Thumbnail.URL = item.Snippet.Thumbnails["high"].URL
		e.Author.Name = item.Snippet.VideoOwnerChannelTitle
		e.Author.URI = "https://www.youtube.com/channel/" + item.Snippet.VideoOwnerChannelID
		entries = append(entries, e)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Published.After(entries[j].Published)
	})
	return entries, nil
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
