// Package proc provided the primary blocking loop
// updating from sources and making feeds
package proc

import (
	"context"
	"regexp"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/syncs"

	"github.com/umputun/feed-master/app/feed"
	"github.com/umputun/feed-master/app/youtube"
)

// TelegramNotif is interface to send messages to telegram
type TelegramNotif interface {
	Send(chanID string, item feed.Item) error
}

// TwitterNotif is interface to send message to twitter
type TwitterNotif interface {
	Send(item feed.Item) error
}

// Processor is a feed reader and store writer
type Processor struct {
	Conf          *Conf
	Store         *BoltDB
	TelegramNotif TelegramNotif
	TwitterNotif  TwitterNotif
}

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
		DlTemplate     string                `yaml:"dl_template"`
		BaseChanURL    string                `yaml:"base_chan_url"`
		Channels       []youtube.ChannelInfo `yaml:"channels"`
		BaseURL        string                `yaml:"base_url"`
		UpdateInterval time.Duration         `yaml:"update"`
		MaxItems       int                   `yaml:"max_per_channel"`
		FilesLocation  string                `yaml:"files_location"`
	} `yaml:"youtube"`
}

// Feed defines config section for a feed~
type Feed struct {
	Title           string `yaml:"title"`
	Description     string `yaml:"description"`
	Link            string `yaml:"link"`
	Image           string `yaml:"image"`
	Language        string `yaml:"language"`
	TelegramChannel string `yaml:"telegram_channel"`
	Filter          Filter `yaml:"filter"`
	Sources         []struct {
		Name string `yaml:"name"`
		URL  string `yaml:"url"`
	} `yaml:"sources"`
	ExtendDateTitle string `yaml:"ext_date"`
}

// Filter defines feed section for a feed filter~
type Filter struct {
	Title string `yaml:"title"`
}

// YTChannel defines youtube channel config
type YTChannel struct {
	ID   string
	Name string
}

// Do activates loop of goroutine for each feed, concurrency limited by p.Conf.Concurrent
func (p *Processor) Do() {
	log.Printf("[INFO] activate processor, feeds=%d, %+v", len(p.Conf.Feeds), p.Conf)
	p.setDefaults()

	for {
		swg := syncs.NewSizedGroup(p.Conf.System.Concurrent, syncs.Preemptive)
		for name, fm := range p.Conf.Feeds {
			for _, src := range fm.Sources {
				name, src, fm := name, src, fm
				swg.Go(func(context.Context) {
					p.feed(name, src.URL, fm.TelegramChannel, p.Conf.System.MaxItems, fm.Filter)
				})
			}
			// keep up to MaxKeepInDB items in bucket
			if removed, err := p.Store.removeOld(name, p.Conf.System.MaxKeepInDB); err == nil {
				if removed > 0 {
					log.Printf("[DEBUG] removed %d from %s", removed, name)
				}
			} else {
				log.Printf("[WARN] failed to remove, %v", err)
			}
		}
		swg.Wait()
		log.Printf("[DEBUG] refresh completed")
		time.Sleep(p.Conf.System.UpdateInterval)
	}
}

func (p *Processor) feed(name, url, telegramChannel string, max int, filter Filter) {
	rss, err := feed.Parse(url)
	if err != nil {
		log.Printf("[WARN] failed to parse %s, %v", url, err)
		return
	}

	// up to MaxItems (5) items from each feed
	upto := max
	if len(rss.ItemList) <= max {
		upto = len(rss.ItemList)
	}

	for _, item := range rss.ItemList[:upto] {
		// skip 1y and older
		if item.DT.Before(time.Now().AddDate(-1, 0, 0)) {
			continue
		}

		skip, err := filter.skip(item)
		if err != nil {
			log.Printf("[WARN] failed to filter %s (%s) to %s, save as is, %v", item.GUID, item.PubDate, name, err)
		}
		if skip {
			item.Junk = true
			log.Printf("[INFO] filtered %s (%s), %s %s", item.GUID, item.PubDate, name, item.Title)
		}

		created, err := p.Store.Save(name, item)
		if err != nil {
			log.Printf("[WARN] failed to save %s (%s) to %s, %v", item.GUID, item.PubDate, name, err)
		}

		if !created {
			return
		}

		if err := p.TelegramNotif.Send(telegramChannel, item); err != nil {
			log.Printf("[WARN] failed to send telegram message, url=%s to channel=%s, %v",
				item.Enclosure.URL, telegramChannel, err)
		}

		if err := p.TwitterNotif.Send(item); err != nil {
			log.Printf("[WARN] failed send twitter message, url=%s, %v", item.Enclosure.URL, err)
		}
	}
}

func (p *Processor) setDefaults() {
	if p.Conf.System.Concurrent == 0 {
		p.Conf.System.Concurrent = 8
	}
	if p.Conf.System.MaxItems == 0 {
		p.Conf.System.MaxItems = 5
	}
	if p.Conf.System.MaxTotal == 0 {
		p.Conf.System.MaxTotal = 100
	}
	if p.Conf.System.MaxKeepInDB == 0 {
		p.Conf.System.MaxKeepInDB = 5000
	}
	if p.Conf.System.UpdateInterval == 0 {
		p.Conf.System.UpdateInterval = time.Minute * 5
	}
}

func (filter *Filter) skip(item feed.Item) (bool, error) {
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
