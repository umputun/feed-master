package proc

import (
	"log"
	"time"

	"github.com/remeh/sizedwaitgroup"

	"github.com/umputun/feed-master/app/feed"
)

// Processor is a feed reader and store writer
type Processor struct {
	Conf  *Conf
	Store *BoltDB
}

// Conf for feeds config yml
type Conf struct {
	Feeds map[string]struct {
		Title       string `yaml:"title"`
		Description string `yaml:"description"`
		Link        string `yaml:"link"`
		Image       string `yaml:"image"`
		Language    string `yaml:"language"`
		Sources     []struct {
			Name string `yaml:"name"`
			URL  string `yaml:"url"`
		} `yaml:"sources"`
	} `yaml:"feeds"`

	UpdateInterval int    `yaml:"update"`
	MaxItems       int    `yaml:"max_per_feed"`
	MaxTotal       int    `yaml:"max_total"`
	MaxKeepInDB    int    `yaml:"max_keep"`
	BaseURL        string `yaml:"base_url"`
	Concurrent     int    `yaml:"concurrent"`
}

// Do activates loop of goroutine for each feed, concurrency limited by p.Conf.Concurrent
func (p *Processor) Do() {
	log.Printf("[INFO] activate processor, fms=%d, %+v", len(p.Conf.Feeds), p.Conf)
	p.setDefaults()

	for {
		swg := sizedwaitgroup.New(p.Conf.Concurrent)
		for name, fm := range p.Conf.Feeds {
			for _, src := range fm.Sources {
				swg.Add()
				go p.feed(name, src.URL, p.Conf.MaxItems, &swg)
			}
			// keep up to MaxKeepInDB items in bucket
			if removed, err := p.Store.removeOld(name, p.Conf.MaxKeepInDB); err == nil {
				if removed > 0 {
					log.Printf("[DEBUG] removed %d from %s", removed, name)
				}
			} else {
				log.Printf("[WARN] failed to remove, %v", err)
			}
		}
		swg.Wait()
		log.Printf("[DEBUG] refresh completed")
		time.Sleep(time.Duration(p.Conf.UpdateInterval) * time.Second)
	}
}

func (p *Processor) feed(name string, url string, max int, swg *sizedwaitgroup.SizedWaitGroup) {
	defer func() {
		swg.Done()
	}()

	rss, err := feed.Parse(url)
	if err != nil {
		log.Printf("[WARN] failed to parse %s, %v", url, err)
		return
	}

	// up to 5 items from each feed
	upto := max
	if len(rss.ItemList) <= max {
		upto = len(rss.ItemList)
	}

	for _, item := range rss.ItemList[:upto] {
		// skip 1y and older
		if item.DT.Before(time.Now().AddDate(-1, 0, 0)) {
			continue
		}

		if err := p.Store.Save(name, item); err != nil {
			log.Printf("[WARN] failed to save %s (%s) to %s, %v", item.GUID, item.PubDate, name, err)
		}
	}
}

func (p *Processor) setDefaults() {
	if p.Conf.Concurrent == 0 {
		p.Conf.Concurrent = 8
	}
	if p.Conf.MaxItems == 0 {
		p.Conf.MaxItems = 5
	}
	if p.Conf.MaxTotal == 0 {
		p.Conf.MaxTotal = 100
	}
	if p.Conf.MaxKeepInDB == 0 {
		p.Conf.MaxKeepInDB = 5000
	}
	if p.Conf.UpdateInterval == 0 {
		p.Conf.UpdateInterval = 600
	}
}
