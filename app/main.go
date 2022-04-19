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

	log "github.com/go-pkgz/lgr"
	"github.com/google/uuid"
	"github.com/jessevdk/go-flags"
	bolt "go.etcd.io/bbolt"

	"github.com/umputun/feed-master/app/duration"

	"github.com/umputun/feed-master/app/config"

	"github.com/umputun/feed-master/app/api"
	rssfeed "github.com/umputun/feed-master/app/feed"
	"github.com/umputun/feed-master/app/proc"
	"github.com/umputun/feed-master/app/youtube"
	ytfeed "github.com/umputun/feed-master/app/youtube/feed"
	"github.com/umputun/feed-master/app/youtube/store"
)

type options struct {
	Port int    `short:"p" long:"port" env:"FM_PORT" description:"port to listen" default:"8080"`
	Conf string `short:"f" long:"conf" env:"FM_CONF" default:"feed-master.yml" description:"config file (yml)"`
}

var revision = "local"

func main() {
	fmt.Printf("feed-master %s\n", revision)
	var opts options
	if _, err := flags.Parse(&opts); err != nil {
		os.Exit(1)
	}

	var conf = &config.Conf{}
	var err error
	conf, err = config.Load(opts.Conf)
	if err != nil {
		log.Fatalf("[ERROR] can't load config %s, %v", opts.Conf, err)
	}
	setupLog(conf.System.Dbg)

	db, err := makeBoltDB(conf.System.DB)
	if err != nil {
		log.Fatalf("[ERROR] can't open db %s, %v", conf.System.DB, err)
	}
	procStore := &proc.BoltDB{DB: db}

	telegramNotif, err := proc.NewTelegramClient(conf.Telegram.Token, conf.Telegram.Server, conf.Telegram.Timeout,
		&duration.Service{})
	if err != nil {
		log.Fatalf("[ERROR] failed to initialize telegram client %s, %v", conf.Telegram.Token, err)
	}

	p := &proc.Processor{Conf: conf, Store: procStore, TelegramNotif: telegramNotif, TwitterNotif: makeTwitter(*conf)}
	go p.Do()

	var ytSvc youtube.Service
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

		ytSvc = youtube.Service{
			Feeds:          conf.YouTube.Channels,
			Downloader:     dwnl,
			ChannelService: &fd,
			Store:          &store.BoltDB{DB: db, Channels: channels},
			CheckDuration:  conf.YouTube.UpdateInterval,
			KeepPerChannel: conf.YouTube.MaxItems,
			RootURL:        conf.YouTube.BaseURL,
			RSSFileStore: youtube.RSSFileStore{
				Location: conf.YouTube.RSSLocation,
				Enabled:  conf.YouTube.RSSLocation != "",
			},
			DurationService: &duration.Service{},
		}
		go func() {
			if err := ytSvc.Do(context.TODO()); err != nil {
				log.Printf("[ERROR] youtube processor failed: %v", err)
			}
		}()
	}

	if conf.System.AdminPasswd == "" {
		log.Printf("[WARN] admin password is not set, protected endpoints are disabled")
		conf.System.AdminPasswd = uuid.New().String() // generate random (uuid) password
	}

	server := api.Server{
		Version:     revision,
		Conf:        *conf,
		Store:       procStore,
		YoutubeSvc:  &ytSvc,
		AdminPasswd: conf.System.AdminPasswd,
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

func makeTwitter(conf config.Conf) *proc.TwitterClient {
	twitterFmtFn := func(item rssfeed.Item) string {
		b1 := bytes.Buffer{}
		if err := template.Must(template.New("twi").Parse(conf.Twitter.Template)).Execute(&b1, item); err != nil { // nolint
			// template failed to parse record, backup predefined format
			return fmt.Sprintf("%s - %s", item.Title, item.Link)
		}
		return strings.ReplaceAll(proc.CleanText(b1.String(), 280), `\n`, "\n") // \n in template
	}

	twiAuth := proc.TwitterAuth{
		ConsumerKey:    conf.Twitter.ConsumerKey,
		ConsumerSecret: conf.Twitter.ConsumerSecret,
		AccessToken:    conf.Twitter.AccessToken,
		AccessSecret:   conf.Twitter.AccessSecret,
	}

	return proc.NewTwitterClient(twiAuth, twitterFmtFn)
}

func setupLog(dbg bool) {
	if dbg {
		log.Setup(log.Debug, log.CallerFile, log.Msec, log.LevelBraces)
		return
	}
	log.Setup(log.Msec, log.LevelBraces)
}
