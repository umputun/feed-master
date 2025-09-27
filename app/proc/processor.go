// Package proc provided the primary blocking loop
// updating from sources and making feeds
package proc

import (
	"context"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/repeater"
	"github.com/go-pkgz/syncs"

	"github.com/umputun/feed-master/app/config"
	"github.com/umputun/feed-master/app/feed"
)

//go:generate moq -out mocks/telegram_notif.go -pkg mocks -skip-ensure -fmt goimports . TelegramNotif
//go:generate moq -out mocks/twitter_notif.go -pkg mocks -skip-ensure -fmt goimports . TwitterNotif

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
func (p *Processor) Do(ctx context.Context) error {
	log.Printf("[INFO] activate processor, feeds=%d, %+v", len(p.Conf.Feeds), p.Conf)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			p.processFeeds(ctx)
		}
	}
}

func (p *Processor) processFeeds(ctx context.Context) {
	log.Printf("[DEBUG] refresh started")
	swg := syncs.NewSizedGroup(p.Conf.System.Concurrent, syncs.Preemptive, syncs.Context(ctx))
	for name, fm := range p.Conf.Feeds {
		for _, src := range fm.Sources {
			name, src, fm := name, src, fm
			swg.Go(func(context.Context) {
				p.processFeed(name, src.URL, fm.TelegramChannel, p.Conf.System.MaxItems, fm.Filter)
			})
		}
	}
	swg.Wait()
	log.Printf("[DEBUG] refresh completed")
	time.Sleep(p.Conf.System.UpdateInterval)
}

func (p *Processor) processFeed(name, url, telegramChannel string, maximum int, filter config.Filter) {
	rss, err := feed.Parse(url)
	if err != nil {
		log.Printf("[WARN] failed to parse %s, %v", url, err)
		return
	}

	// up to MaxItems (5) items from each feed
	upto := maximum
	if len(rss.ItemList) <= maximum {
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

		rptr := repeater.NewDefault(3, 5*time.Second)
		attemptNum := 0
		err = rptr.Do(context.Background(), func() error {
			attemptNum++
			startTime := time.Now()
			log.Printf("[DEBUG] sending telegram message (attempt %d/3): title=%q, size=%d bytes, url=%s to channel=%s",
				attemptNum, item.Title, item.Enclosure.Length, item.Enclosure.URL, telegramChannel)

			if e := p.TelegramNotif.Send(telegramChannel, item); e != nil {
				elapsed := time.Since(startTime)
				log.Printf("[WARN] failed attempt %d/3 to send telegram message after %v: title=%q, size=%d bytes, url=%s to channel=%s, error=%v",
					attemptNum, elapsed, item.Title, item.Enclosure.Length, item.Enclosure.URL, telegramChannel, e)
				return e
			}

			elapsed := time.Since(startTime)
			log.Printf("[INFO] successfully sent telegram message in %v: title=%q, size=%d bytes",
				elapsed, item.Title, item.Enclosure.Length)
			return nil
		})
		if err != nil {
			log.Printf("[WARN] failed to send telegram message after 3 attempts: title=%q, size=%d bytes, url=%s to channel=%s, final_error=%v",
				item.Title, item.Enclosure.Length, item.Enclosure.URL, telegramChannel, err)
		}

		if err := p.TwitterNotif.Send(item); err != nil {
			log.Printf("[WARN] failed send twitter message, url=%s, %v", item.Enclosure.URL, err)
		}
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
