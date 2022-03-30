// Package proc provided the primary blocking loop
// updating from sources and making feeds
package proc

import (
	"context"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/syncs"

	"github.com/umputun/feed-master/app/config"
	"github.com/umputun/feed-master/app/feed"
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
	Conf          *config.Conf
	Store         *BoltDB
	TelegramNotif TelegramNotif
	TwitterNotif  TwitterNotif
}

// Do activate loop of goroutine for each feed, concurrency limited by p.Conf.Concurrent
func (p *Processor) Do() {
	log.Printf("[INFO] activate processor, feeds=%d, %+v", len(p.Conf.Feeds), p.Conf)
	config.SetDefaults(p.Conf)

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

func (p *Processor) feed(name, url, telegramChannel string, max int, filter config.Filter) {
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

		skip, err := filter.Skip(item)
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

		// don't attempt to send anything if the entry was already saved
		// or in case it was filtered out
		if !created || item.Junk {
			continue
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
