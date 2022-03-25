package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"text/template"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/jessevdk/go-flags"
	bolt "go.etcd.io/bbolt"
	"gopkg.in/yaml.v2"

	"github.com/umputun/feed-master/app/youtube"
	"github.com/umputun/feed-master/app/youtube/channel"
	"github.com/umputun/feed-master/app/youtube/store"

	"github.com/umputun/feed-master/app/api"
	"github.com/umputun/feed-master/app/feed"
	"github.com/umputun/feed-master/app/proc"
)

type options struct {
	DB   string `short:"c" long:"db" env:"FM_DB" default:"var/feed-master.bdb" description:"bolt db file"`
	Conf string `short:"f" long:"conf" env:"FM_CONF" default:"feed-master.yml" description:"config file (yml)"`

	// single feed overrides
	Feed            string        `long:"feed" env:"FM_FEED" description:"single feed, overrides config"`
	TelegramChannel string        `long:"telegram_chan" env:"TELEGRAM_CHAN" description:"single telegram channel, overrides config"`
	UpdateInterval  time.Duration `long:"update-interval" env:"UPDATE_INTERVAL" default:"1m" description:"update interval, overrides config"`

	TelegramServer        string        `long:"telegram_server" env:"TELEGRAM_SERVER" default:"https://api.telegram.org" description:"telegram bot api server"`
	TelegramToken         string        `long:"telegram_token" env:"TELEGRAM_TOKEN" description:"telegram token"`
	TelegramTimeout       time.Duration `long:"telegram_timeout" env:"TELEGRAM_TIMEOUT" default:"1m" description:"telegram timeout"`
	TwitterConsumerKey    string        `long:"consumer-key" env:"TWI_CONSUMER_KEY" description:"twitter consumer key"`
	TwitterConsumerSecret string        `long:"consumer-secret" env:"TWI_CONSUMER_SECRET" description:"twitter consumer secret"`
	TwitterAccessToken    string        `long:"access-token" env:"TWI_ACCESS_TOKEN" description:"twitter access token"`
	TwitterAccessSecret   string        `long:"access-secret" env:"TWI_ACCESS_SECRET" description:"twitter access secret"`
	TwitterTemplate       string        `long:"template" env:"TEMPLATE" default:"{{.Title}} - {{.Link}}" description:"twitter message template"`

	YtLocation string `long:"yt-location" env:"YT_LOCATION" default:"var/yt" description:"path to youtube download location"`

	Dbg bool `long:"dbg" env:"DEBUG" description:"debug mode"`
}

var revision = "local"

func main() {
	fmt.Printf("feed-master %s\n", revision)
	var opts options
	if _, err := flags.Parse(&opts); err != nil {
		os.Exit(1)
	}
	setupLog(opts.Dbg)

	var conf = &proc.Conf{}
	if opts.Feed != "" { // single feed (no config) mode
		conf = singleFeedConf(opts.Feed, opts.TelegramChannel, opts.UpdateInterval)
	}

	var err error
	if opts.Feed == "" {
		conf, err = loadConfig(opts.Conf)
		if err != nil {
			log.Fatalf("[ERROR] can't load config %s, %v", opts.Conf, err)
		}
	}

	db, err := makeBoltDB(opts.DB)
	if err != nil {
		log.Fatalf("[ERROR] can't open db %s, %v", opts.DB, err)
	}
	procStore := &proc.BoltDB{DB: db}

	telegramNotif, err := proc.NewTelegramClient(opts.TelegramToken, opts.TelegramServer, opts.TelegramTimeout)
	if err != nil {
		log.Fatalf("[ERROR] failed to initialize telegram client %s, %v", opts.TelegramToken, err)
	}

	p := &proc.Processor{Conf: conf, Store: procStore, TelegramNotif: telegramNotif, TwitterNotif: makeTwitter(opts)}
	go p.Do()

	var ytSvc youtube.Service
	if len(conf.YouTube.Channels) > 0 {
		log.Printf("[INFO] starting youtube processor for %d channels", len(conf.YouTube.Channels))
		dwnl := channel.NewDownloader(conf.YouTube.DlTemplate, log.ToWriter(log.Default(), "DEBUG"), opts.YtLocation)
		fd := channel.Feed{Client: &http.Client{Timeout: 10 * time.Second}, BaseURL: conf.YouTube.BaseChanURL}
		ytSvc = youtube.Service{
			Channels:       conf.YouTube.Channels,
			Downloader:     dwnl,
			ChannelService: &fd,
			Store:          &store.BoltDB{DB: db},
			CheckDuration:  conf.YouTube.UpdateInterval,
			KeepPerChannel: conf.YouTube.MaxItems,
			RootURL:        conf.YouTube.BaseURL,
			RSSFileStore: youtube.RSSFileStore{
				Location: conf.YouTube.RSSLocation,
				Enabled:  conf.YouTube.RSSLocation != "",
			},
		}
		go func() {
			if err := ytSvc.Do(context.TODO()); err != nil {
				log.Printf("[ERROR] youtube processor failed: %v", err)
			}
		}()
	}

	server := api.Server{
		Version:    revision,
		Conf:       *conf,
		Store:      procStore,
		YoutubeSvc: &ytSvc,
	}
	server.Run(8080)
}

func singleFeedConf(feedURL, ch string, updateInterval time.Duration) *proc.Conf {
	conf := proc.Conf{}
	f := proc.Feed{
		TelegramChannel: ch,
		Sources: []struct {
			Name string `yaml:"name"`
			URL  string `yaml:"url"`
		}{
			{Name: "auto", URL: feedURL},
		},
	}
	conf.Feeds = map[string]proc.Feed{"auto": f}
	conf.System.UpdateInterval = updateInterval
	return &conf
}

func makeBoltDB(dbFile string) (*bolt.DB, error) {
	log.Printf("[INFO] bolt (persistent) store, %s", dbFile)
	if dbFile == "" {
		return nil, fmt.Errorf("empty db")
	}
	if err := os.MkdirAll(path.Dir(dbFile), 0o700); err != nil {
		return nil, err
	}
	db, err := bolt.Open(dbFile, 0o600, &bolt.Options{Timeout: 1 * time.Second}) // nolint
	if err != nil {
		return nil, err
	}

	return db, err
}

func makeTwitter(opts options) *proc.TwitterClient {
	twitterFmtFn := func(item feed.Item) string {
		b1 := bytes.Buffer{}
		if err := template.Must(template.New("twi").Parse(opts.TwitterTemplate)).Execute(&b1, item); err != nil { // nolint
			// template failed to parse record, backup predefined format
			return fmt.Sprintf("%s - %s", item.Title, item.Link)
		}
		return strings.ReplaceAll(proc.CleanText(b1.String(), 280), `\n`, "\n") // \n in template
	}

	twiAuth := proc.TwitterAuth{
		ConsumerKey:    opts.TwitterConsumerKey,
		ConsumerSecret: opts.TwitterConsumerSecret,
		AccessToken:    opts.TwitterAccessToken,
		AccessSecret:   opts.TwitterAccessSecret,
	}

	return proc.NewTwitterClient(twiAuth, twitterFmtFn)
}

func loadConfig(fname string) (res *proc.Conf, err error) {
	res = &proc.Conf{}
	data, err := ioutil.ReadFile(fname) // nolint
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, res); err != nil {
		return nil, err
	}

	return res, nil
}

func setupLog(dbg bool) {
	if dbg {
		log.Setup(log.Debug, log.CallerFile, log.Msec, log.LevelBraces)
		return
	}
	log.Setup(log.Msec, log.LevelBraces)
}
