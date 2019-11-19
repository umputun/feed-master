package main

import (
	"fmt"
	"io/ioutil"
	"os"

	log "github.com/go-pkgz/lgr"
	"github.com/jessevdk/go-flags"
	"gopkg.in/yaml.v2"

	"github.com/umputun/feed-master/app/api"
	"github.com/umputun/feed-master/app/proc"
)

var opts struct {
	DB   string `short:"c" long:"db" env:"FM_DB" default:"var/feed-master.bdb" description:"bolt db file"`
	Conf string `short:"f" long:"conf" env:"FM_CONF" default:"feed-master.yml" description:"config file (yml)"`
	Dbg  bool   `long:"dbg" env:"DEBUG" description:"debug mode"`
	TG   string `long:"telegram_token" env:"TELEGRAM_TOKEN" description:"Telegram token"`
}

var revision = "local"

func main() {
	fmt.Printf("feed-master %s\n", revision)
	if _, err := flags.Parse(&opts); err != nil {
		os.Exit(1)
	}
	setupLog(opts.Dbg)

	conf, err := loadConfig(opts.Conf)
	if err != nil {
		log.Fatalf("[ERROR] can't load config %s, %v", opts.Conf, err)
	}
	db, err := proc.NewBoltDB(opts.DB)
	if err != nil {
		log.Fatalf("[ERROR] can't open db %s, %v", opts.DB, err)
	}
	tg, err := proc.NewTelegramClient(opts.TG)
	if err != nil {
		log.Fatalf("[ERROR] failed initilization telegram client %s, %v", opts.TG, err)
	}

	p := &proc.Processor{Conf: conf, Store: db, Notification: tg}
	go p.Do()

	server := api.Server{
		Version: revision,
		Conf:    *conf,
		Store:   db,
	}
	server.Run(8080)
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
