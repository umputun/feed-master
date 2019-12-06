// Package proc provided the primary blockin loop
// updating from sources and making feeds
package proc

import (
	"context"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/syncs"

	"github.com/umputun/feed-master/app/feed"
)

// Notification is interface implemented send message
type Notification interface {
	Send(string, feed.Item) error
}

// Processor is a feed reader and store writer
type Processor struct {
	Conf         *Conf
	Store        *BoltDB
	Notification Notification
}

// Conf for feeds config yml
type Conf struct {
	Feeds map[string]struct {
		Title           string `yaml:"title"`
		Description     string `yaml:"description"`
		Link            string `yaml:"link"`
		Image           string `yaml:"image"`
		Language        string `yaml:"language"`
		TelegramChannel string `yaml:"telegram_channel"`
		Sources         []struct {
			Name string `yaml:"name"`
			URL  string `yaml:"url"`
		} `yaml:"sources"`
	} `yaml:"feeds"`

	System struct {
		UpdateInterval time.Duration `yaml:"update"`
		MaxItems       int           `yaml:"max_per_feed"`
		MaxTotal       int           `yaml:"max_total"`
		MaxKeepInDB    int           `yaml:"max_keep"`
		Concurrent     int           `yaml:"concurrent"`
		BaseURL        string        `yaml:"base_url"`
	} `yaml:"system"`
}

// Do activates loop of goroutine for each feed, concurrency limited by p.Conf.Concurrent
func (p *Processor) Do() {
	log.Printf("[INFO] activate processor, feeds=%d, %+v", len(p.Conf.Feeds), p.Conf)
	p.setDefaults()

	for {
		swg := syncs.NewSizedGroup(p.Conf.System.Concurrent, syncs.Preemptive)
		for name, fm := range p.Conf.Feeds {
			for _, src := range fm.Sources {
				name, src, tgChan := name, src, fm.TelegramChannel
				swg.Go(func(context.Context) {
					p.feed(name, src.URL, tgChan, p.Conf.System.MaxItems)
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

func (p *Processor) feed(name, url, telegramChannel string, max int) {

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

		created, err := p.Store.Save(name, item)
		if err != nil {
			log.Printf("[WARN] failed to save %s (%s) to %s, %v", item.GUID, item.PubDate, name, err)
		}

		if !created {
			return
		}

		if err := p.Notification.Send(telegramChannel, item); err != nil {
			log.Printf("[WARN] failed send telegram message, url=%s to channel=%s, %v", item.Enclosure.URL, telegramChannel, err)
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
