package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"text/template"
	"time"

	"github.com/ChimeraCoder/anaconda"

	log "github.com/go-pkgz/lgr"
	"github.com/google/uuid"
	"github.com/jessevdk/go-flags"
	bolt "go.etcd.io/bbolt"

	"github.com/umputun/feed-master/app/api"
	"github.com/umputun/feed-master/app/config"
	"github.com/umputun/feed-master/app/duration"
	rssfeed "github.com/umputun/feed-master/app/feed"
	"github.com/umputun/feed-master/app/proc"
	"github.com/umputun/feed-master/app/youtube"
	ytfeed "github.com/umputun/feed-master/app/youtube/feed"
	"github.com/umputun/feed-master/app/youtube/store"
)

type options struct {
	Port int    `short:"p" long:"port" description:"port to listen" default:"8080"`
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

	AdminPasswd string `long:"admin-passwd" env:"ADMIN_PASSWD" description:"admin password for protected endpoints"`

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

	var conf = &config.Conf{}
	if opts.Feed != "" { // single feed (no config) mode
		conf = config.SingleFeed(opts.Feed, opts.TelegramChannel, opts.UpdateInterval)
	}

	var err error
	if opts.Feed == "" {
		conf, err = config.Load(opts.Conf)
		if err != nil {
			log.Fatalf("[ERROR] can't load config %s, %v", opts.Conf, err)
		}
	}

	db, err := makeBoltDB(opts.DB)
	if err != nil {
		log.Fatalf("[ERROR] can't open db %s, %v", opts.DB, err)
	}
	procStore := &proc.BoltDB{DB: db}

	telegramNotif, err := proc.NewTelegramClient(opts.TelegramToken, opts.TelegramServer, opts.TelegramTimeout,
		&duration.Service{}, &proc.TelegramSenderImpl{})
	if err != nil {
		log.Fatalf("[ERROR] failed to initialize telegram client %s, %v", opts.TelegramToken, err)
	}

	p := &proc.Processor{Conf: conf, Store: procStore, TelegramNotif: telegramNotif, TwitterNotif: makeTwitter(opts)}
	go func() {
		if err := p.Do(context.Background()); err != nil {
			log.Printf("[ERROR] processor failed: %v", err)
		}
	}()

	var ytSvc youtube.Service
	var ytStore *store.BoltDB
	if len(conf.YouTube.Channels) > 0 {
		log.Printf("[INFO] starting youtube processor for %d channels", len(conf.YouTube.Channels))
		outWr := log.ToWriter(log.Default(), "DEBUG")
		errWr := log.ToWriter(log.Default(), "INFO")
		dwnl := ytfeed.NewDownloader(conf.YouTube.DlTemplate, outWr, errWr, conf.YouTube.FilesLocation)
		fd := ytfeed.Feed{Client: &http.Client{Timeout: 10 * time.Second},
			ChannelBaseURL: conf.YouTube.BaseChanURL, PlaylistBaseURL: conf.YouTube.BasePlaylistURL}

		channels := []string{}
		for _, c := range conf.YouTube.Channels {
			channels = append(channels, c.ID)
		}
		log.Printf("[DEBUG] buckets for youtube store: %s", strings.Join(channels, ", "))

		ytStore = &store.BoltDB{DB: db, Channels: channels}
		ytSvc = youtube.Service{
			Feeds:          conf.YouTube.Channels,
			Downloader:     dwnl,
			ChannelService: &fd,
			Store:          ytStore,
			CheckDuration:  conf.YouTube.UpdateInterval,
			KeepPerChannel: conf.YouTube.MaxItems,
			RootURL:        conf.YouTube.BaseURL,
			RSSFileStore: youtube.RSSFileStore{
				Location: conf.YouTube.RSSLocation,
				Enabled:  conf.YouTube.RSSLocation != "",
			},
			DurationService: &duration.Service{},
			SkipShorts:      conf.YouTube.SkipShorts,
		}
		if conf.YouTube.YtDlpUpdate.Interval > 0 {
			log.Printf("[INFO] yt-dlp updater enabled, interval %s", conf.YouTube.YtDlpUpdate.Interval)
			ytSvc.YtDlpUpdCommand = conf.YouTube.YtDlpUpdate.Command
			ytSvc.YtDlpUpdDuration = conf.YouTube.YtDlpUpdate.Interval
		} else {
			log.Printf("[INFO] yt-dlp updater is disabled")
		}

		go func() {
			if conf.YouTube.DisableUpdates {
				log.Printf("[INFO] youtube updates are disabled")
				return
			}
			if err := ytSvc.Do(context.TODO()); err != nil {
				log.Printf("[ERROR] youtube processor failed: %v", err)
			}
		}()
	}

	if opts.AdminPasswd == "" {
		log.Printf("[WARN] admin password is not set, protected endpoints are disabled")
		opts.AdminPasswd = uuid.New().String() // generate random (uuid) password
	}

	server := api.Server{
		Version:      revision,
		Conf:         *conf,
		Store:        procStore,
		YoutubeStore: ytStore,
		YoutubeSvc:   &ytSvc,
		AdminPasswd:  opts.AdminPasswd,
	}
	server.Run(context.Background(), opts.Port)
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
	twitterFmtFn := func(item rssfeed.Item) string {
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

	twitPoster := anaconda.NewTwitterApiWithCredentials(twiAuth.AccessToken, twiAuth.AccessSecret, twiAuth.ConsumerKey, twiAuth.ConsumerSecret)

	return proc.NewTwitterClient(twiAuth, twitterFmtFn, twitPoster)
}

func setupLog(dbg bool) {
	if dbg {
		log.Setup(log.Debug, log.CallerFile, log.Msec, log.LevelBraces)
		return
	}
	log.Setup(log.Msec, log.LevelBraces)
}
