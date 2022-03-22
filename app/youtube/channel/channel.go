// Package channel provides youtube's feed parser and downloader.
package channel

import (
	"html/template"
	"time"
)

// Entry represents a YouTube channel entry.
type Entry struct {
	ChannelID string `xml:"http://www.youtube.com/xml/schemas/2015 channelId"`
	VideoID   string `xml:"http://www.youtube.com/xml/schemas/2015 videoId"`
	Title     string `xml:"title"`
	Link      struct {
		Href string `xml:"href,attr"`
	} `xml:"link"`
	Published time.Time `xml:"published"`

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
	File string
}
