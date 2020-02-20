package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"text/template"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/jessevdk/go-flags"
	"gopkg.in/yaml.v2"

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

	TelegramToken         string        `long:"telegram_token" env:"TELEGRAM_TOKEN" description:"telegram token"`
	TelegramTimeout       time.Duration `long:"telegram_timeout" env:"TELEGRAM_TIMEOUT" default:"1m" description:"telegram timeout"`
	TwitterConsumerKey    string        `long:"consumer-key" env:"TWI_CONSUMER_KEY" description:"twitter consumer key"`
	TwitterConsumerSecret string        `long:"consumer-secret" env:"TWI_CONSUMER_SECRET" description:"twitter consumer secret"`
	TwitterAccessToken    string        `long:"access-token" env:"TWI_ACCESS_TOKEN" description:"twitter access token"`
	TwitterAccessSecret   string        `long:"access-secret" env:"TWI_ACCESS_SECRET" description:"twitter access secret"`
	TwitterTemplate       string        `long:"template" env:"TEMPLATE" default:"{{.Title}} - {{.Link}}" description:"twitter message template"`

	Uploader struct {
		Enabled      bool   `long:"enabled" env:"UPLOADER_ENABLED" description:"Enables experimental Telegram large files uploader"`
		PathToScript string `long:"path" env:"PATH_TO_SCRIPT" description:"Path to Python uploader script"`
		APIID        string `long:"api_id" env:"API_ID" description:"Telegram APP API ID in format like 0000000"`
		APIHash      string `long:"api_hash" env:"API_HASH" description:"Telegram APP API Hash in format like 0123456789acbdefghijklmnopqrstuw"`
		Session      string `long:"session" env:"SESSION" description:"Path to Telegram client session file (created by uploader/auth.py script)"`
	} `group:"uploader" namespace:"uploader" env-namespace:"UPLOADER"`

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

	db, err := proc.NewBoltDB(opts.DB)
	if err != nil {
		log.Fatalf("[ERROR] can't open db %s, %v", opts.DB, err)
	}

	telegramNotif, err := proc.NewTelegramClient(
		opts.TelegramToken,
		opts.TelegramTimeout,
		proc.TelegramUploaderConfig{
			Enabled:      opts.Uploader.Enabled,
			PathToScript: opts.Uploader.PathToScript,
			APIID:        opts.Uploader.APIID,
			APIHash:      opts.Uploader.APIHash,
			Session:      opts.Uploader.Session,
		},
	)
	if err != nil {
		log.Fatalf("[ERROR] failed to initialize telegram client %s, %v", opts.TelegramToken, err)
	}

	p := &proc.Processor{Conf: conf, Store: db, TelegramNotif: telegramNotif, TwitterNotif: makeTwitter(opts)}
	go p.Do()

	server := api.Server{
		Version: revision,
		Conf:    *conf,
		Store:   db,
	}
	server.Run(8080)
}

func singleFeedConf(feedURL, channel string, updateInterval time.Duration) *proc.Conf {
	conf := proc.Conf{}
	f := proc.Feed{
		TelegramChannel: channel,
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

func makeTwitter(opts options) *proc.TwitterClient {
	twitterFmtFn := func(item feed.Item) string {
		b1 := bytes.Buffer{}
		if err := template.Must(template.New("twi").Parse(opts.TwitterTemplate)).Execute(&b1, item); err != nil { // nolint
			// template failed to parse record, backup predefined format
			return fmt.Sprintf("%s - %s", item.Title, item.Link)
		}
		return strings.Replace(proc.CleanText(b1.String(), 275), `\n`, "\n", -1) // \n in template
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
