// Package feed describes RSS format and provides parsing
package feed

// based on http://siongui.github.io/2015/03/03/go-parse-web-feed-rss-atom/

import (
	"encoding/xml"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/pkg/errors"
)

// Rss2 feed
type Rss2 struct {
	XMLName        xml.Name        `xml:"rss"`
	Version        string          `xml:"version,attr"`
	NsItunes       string          `xml:"xmlns:itunes,attr"`
	NsMedia        string          `xml:"xmlns:media,attr"`
	Title          string          `xml:"channel>title"`
	Language       string          `xml:"channel>language"`
	Link           string          `xml:"channel>link"`
	Description    string          `xml:"channel>description"`
	PubDate        string          `xml:"channel>pubDate"`
	LastBuildDate  string          `xml:"channel>lastBuildDate"`
	ItunesImage    *ItunesImg      `xml:"channel>itunes:image"`
	MediaThumbnail *MediaThumbnail `xml:"channel>media:thumbnail"`
	ItunesAuthor   string          `xml:"channel>itunes:author"`
	ItunesExplicit string          `xml:"channel>itunes:explicit"`
	ItunesOwner    *ItunesOwner    `xml:"channel>itunes:owner"`
	ItemList       []Item          `xml:"channel>item"`
}

// ItunesImg image element for iTunes
type ItunesImg struct {
	XMLName xml.Name `xml:"itunes:image,omitempty"`
	URL     string   `xml:"href,attr"`
}

// ItunesOwner owner element for iTunes
type ItunesOwner struct {
	Email string `xml:"itunes:email,omitempty"`
	Name  string `xml:"itunes:name,omitempty"`
}

// MediaThumbnail image element for media
type MediaThumbnail struct {
	XMLName xml.Name `xml:"media:thumbnail,omitempty"`
	URL     string   `xml:"url,attr"`
}

// Enclosure element from item
type Enclosure struct {
	URL    string `xml:"url,attr"`
	Length int    `xml:"length,attr"`
	Type   string `xml:"type,attr"`
}

// Atom1 is atom feed
type Atom1 struct {
	XMLName   xml.Name `xml:"http://www.w3.org/2005/Atom feed"`
	Title     string   `xml:"title"`
	Subtitle  string   `xml:"subtitle"`
	ID        string   `xml:"id"`
	Updated   string   `xml:"updated"`
	Rights    string   `xml:"rights"`
	Link      Link     `xml:"link"`
	Author    Author   `xml:"author"`
	EntryList []Entry  `xml:"entry"`
}

// Link element for xml
type Link struct {
	Href string `xml:"href,attr"`
}

// Author element for xml
type Author struct {
	Name  string `xml:"name"`
	Email string `xml:"email"`
}

// Entry from atom
type Entry struct {
	Title     string    `xml:"title"`
	Summary   string    `xml:"summary"`
	Content   string    `xml:"content"`
	ID        string    `xml:"id"`
	Updated   string    `xml:"updated"`
	Link      Link      `xml:"link"`
	Author    Author    `xml:"author"`
	Enclosure Enclosure `xml:"enclosure"`
}

// Parse gets url to rss feed and returns Rss2 items
func Parse(uri string) (result Rss2, err error) {
	client := http.Client{Timeout: time.Minute * 2}
	resp, err := client.Get(uri) // nolint
	if err != nil {
		return result, err
	}
	defer func() {
		if e := resp.Body.Close(); e != nil {
			log.Printf("[WARN] failed to close body, %s", e)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf("non-200 status code %s, url: %s", resp.Status, uri)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, errors.Wrapf(err, "failed to read body, url: %s", uri)
	}
	result, err = parseFeedContent(body)
	if err != nil {
		return Rss2{}, errors.Wrap(err, "parsing error")
	}

	return result.Normalize()
}

func atom1ToRss2(a Atom1) Rss2 {
	r := Rss2{
		Title:       a.Title,
		Link:        a.Link.Href,
		Description: a.Subtitle,
		PubDate:     a.Updated,
	}
	r.ItemList = make([]Item, len(a.EntryList))
	for i, entry := range a.EntryList {
		r.ItemList[i].Title = entry.Title
		r.ItemList[i].Link = entry.Link.Href
		if entry.Content == "" {
			r.ItemList[i].Description = template.HTML(entry.Summary) // nolint
		} else {
			r.ItemList[i].Description = template.HTML(entry.Content) // nolint
		}
	}
	return r
}

const atomErrStr = "expected element type <rss> but have <feed>"

func parseAtom(content []byte) (Rss2, error) {
	a := Atom1{}
	err := xml.Unmarshal(content, &a)
	if err != nil {
		return Rss2{}, errors.Wrap(err, "can't parse atom1")
	}
	return atom1ToRss2(a), nil
}

func parseFeedContent(content []byte) (Rss2, error) {
	v := Rss2{}
	err := xml.Unmarshal(content, &v)
	if err != nil {
		if err.Error() == atomErrStr {
			// try Atom 1.0
			return parseAtom(content)
		}
		return v, errors.Wrap(err, "can't parse feed content")
	}

	if v.Version == "2.0" {
		// RSS 2.0
		v.NsItunes = "http://www.itunes.com/dtds/podcast-1.0.dtd"
		for i := range v.ItemList {
			if v.ItemList[i].Content != "" {
				v.ItemList[i].Description = v.ItemList[i].Content
			}
		}
		return v, nil
	}

	return v, errors.New("not RSS 2.0")
}

// Normalize converts to RFC822 = "02 Jan 06 15:04 MST"
func (rss *Rss2) Normalize() (Rss2, error) {

	dt, err := rss.parseDateTime(rss.LastBuildDate)
	if err != nil {
		log.Printf("[DEBUG] failed to parse LastBuildDate: %v, fallback with PubDate", err)
		dt, err = rss.parseDateTime(rss.PubDate)
	}
	if err == nil {
		rss.PubDate = dt.Format(time.RFC1123Z)
	}

	for i, item := range rss.ItemList {
		if dt, err := rss.parseDateTime(item.PubDate); err == nil {
			rss.ItemList[i].DT = dt
			rss.ItemList[i].PubDate = dt.Format(time.RFC1123Z)
		}
		rss.ItemList[i].Title = strings.ReplaceAll(item.Title, "\n", "")
		rss.ItemList[i].Title = strings.TrimSpace(rss.ItemList[i].Title)
	}
	return *rss, nil
}

func (rss *Rss2) parseDateTime(dt string) (time.Time, error) {
	if dt == "" {
		return time.Now(), fmt.Errorf("can't parse empty date-time")
	}
	if ts, err := time.Parse(time.RFC822, dt); err == nil {
		return ts, nil
	}
	if ts, err := time.Parse(time.RFC822Z, dt); err == nil {
		return ts, nil
	}
	if ts, err := time.Parse(time.RFC1123Z, dt); err == nil {
		return ts, nil
	}
	if ts, err := time.Parse(time.RFC1123, dt); err == nil {
		return ts, nil
	}
	if ts, err := time.Parse("2006-01-02 15:04:05 -0700", dt); err == nil {
		return ts, nil
	}
	if ts, err := time.Parse("2006-01-02T15:04:05-0700", dt); err == nil {
		return ts, nil
	}

	return time.Now(), fmt.Errorf("can't parse timestamp %s", dt)
}
