// Package config provides the configuration support for the application.
package config

import (
	"io/ioutil"
	"regexp"
	"time"

	"github.com/umputun/feed-master/app/feed"
	"gopkg.in/yaml.v2"

	"github.com/umputun/feed-master/app/youtube"
)

// Conf for feeds config yml
type Conf struct {
	Feeds  map[string]Feed `yaml:"feeds"`
	System struct {
		UpdateInterval time.Duration `yaml:"update"`
		MaxItems       int           `yaml:"max_per_feed"`
		MaxTotal       int           `yaml:"max_total"`
		MaxKeepInDB    int           `yaml:"max_keep"`
		Concurrent     int           `yaml:"concurrent"`
		BaseURL        string        `yaml:"base_url"`
	} `yaml:"system"`

	YouTube struct {
		DlTemplate      string             `yaml:"dl_template"`
		BaseChanURL     string             `yaml:"base_chan_url"`
		BasePlaylistURL string             `yaml:"base_playlist_url"`
		Channels        []youtube.FeedInfo `yaml:"channels"`
		BaseURL         string             `yaml:"base_url"`
		UpdateInterval  time.Duration      `yaml:"update"`
		MaxItems        int                `yaml:"max_per_channel"`
		FilesLocation   string             `yaml:"files_location"`
		RSSLocation     string             `yaml:"rss_location"`
	} `yaml:"youtube"`
}

// Source defines config section for source
type Source struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

// Feed defines config section for a feed~
type Feed struct {
	Title           string   `yaml:"title"`
	Description     string   `yaml:"description"`
	Link            string   `yaml:"link"`
	Image           string   `yaml:"image"`
	Language        string   `yaml:"language"`
	TelegramChannel string   `yaml:"telegram_channel"`
	Filter          Filter   `yaml:"filter"`
	Sources         []Source `yaml:"sources"`
	ExtendDateTitle string   `yaml:"ext_date"`
}

// Filter defines feed section for a feed filter~
type Filter struct {
	Title string `yaml:"title"`
}

// Skip items with this regexp
func (filter *Filter) Skip(item feed.Item) (bool, error) {
	if filter.Title != "" {
		matched, err := regexp.MatchString(filter.Title, item.Title)
		if err != nil {
			return matched, err
		}
		if matched {
			return true, err
		}
	}
	return false, nil
}

// YTChannel defines youtube channel config
type YTChannel struct {
	ID   string
	Name string
}

// Load config from file
func Load(fname string) (res *Conf, err error) {
	res = &Conf{}
	data, err := ioutil.ReadFile(fname) // nolint
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, res); err != nil {
		return nil, err
	}

	return res, nil
}

// SingleFeed returns single feed "fake" config for no-config mode
func SingleFeed(feedURL, ch string, updateInterval time.Duration) *Conf {
	conf := Conf{}
	f := Feed{
		TelegramChannel: ch,
		Sources: []Source{
			{Name: "auto", URL: feedURL},
		},
	}
	conf.Feeds = map[string]Feed{"auto": f}
	conf.System.UpdateInterval = updateInterval
	return &conf
}

// SetDefaults sets default values for config
func SetDefaults(c *Conf) {
	if c.System.Concurrent == 0 {
		c.System.Concurrent = 8
	}
	if c.System.MaxItems == 0 {
		c.System.MaxItems = 5
	}
	if c.System.MaxTotal == 0 {
		c.System.MaxTotal = 100
	}
	if c.System.MaxKeepInDB == 0 {
		c.System.MaxKeepInDB = 5000
	}
	if c.System.UpdateInterval == 0 {
		c.System.UpdateInterval = time.Minute * 5
	}
}
