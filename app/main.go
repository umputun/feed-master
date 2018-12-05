package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/logutils"
	"github.com/jessevdk/go-flags"
	"gopkg.in/yaml.v2"

	"github.com/umputun/feed-master/app/feed"
	"github.com/umputun/feed-master/app/proc"
)

var opts struct {
	DB   string `short:"c" long:"db" env:"FM_DB" default:"var/feed-master.bdb" description:"bolt db file"`
	Conf string `short:"f" long:"conf" env:"FM_CONF" default:"feed-master.yml" description:"config file (yml)"`
	Dbg  bool   `long:"dbg" env:"DEBUG" description:"debug mode"`
}

var revision string

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

	p := &proc.Processor{Conf: conf, Store: db}
	go p.Do()

	serveHTTP(db, conf)
}

func serveHTTP(db *proc.BoltDB, conf *proc.Conf) {
	log.Print("[INFO] serve HTTP activated")
	// GET /rss/:name
	http.HandleFunc("/rss/", func(w http.ResponseWriter, r *http.Request) {
		st := time.Now()
		fm := r.URL.Path[len("/rss/"):]
		items, err := db.Load(fm, conf.MaxTotal) //50
		if err != nil {
			w.WriteHeader(404)
			fmt.Fprintf(w, "error: %s", err)
			return
		}

		rss := feed.Rss2{
			Version:       "2.0",
			ItemList:      items,
			Title:         conf.Feeds[fm].Title,
			Description:   conf.Feeds[fm].Description,
			Language:      conf.Feeds[fm].Language,
			Link:          conf.Feeds[fm].Link,
			PubDate:       items[0].PubDate,
			LastBuildDate: time.Now().Format(time.RFC822Z),
			//Image:         &feed.IImage{HREF: fmt.Sprintf("%s/image/%s.png", conf.BaseURL, fm)},
		}
		b, err := xml.MarshalIndent(&rss, "", "  ")
		if err != nil {
			log.Printf("[WARN] failed to marshal rss, %v", err)
			w.WriteHeader(500)
			fmt.Fprintf(w, "error: %s", err)
			return
		}
		log.Printf("[INFO] %s - %v - %s - %s - %s", r.URL.Path, time.Since(st), r.RemoteAddr, r.Referer(), r.UserAgent())
		w.Header().Set("Content-Type", "application/xml; charset=UTF-8")
		fmt.Fprintf(w, "%s", string(b))
	})

	// GET /image/:name
	http.HandleFunc("/image/", func(w http.ResponseWriter, r *http.Request) {
		st := time.Now()
		fm := r.URL.Path[len("/image/"):]
		fm = strings.TrimRight(fm, ".png")
		feedConf, found := conf.Feeds[fm]
		if !found {
			log.Printf("[WARN] failed to load image for %s", fm)
			w.WriteHeader(400)
			fmt.Fprintf(w, "error: no such feed %s", fm)
			return
		}

		b, err := ioutil.ReadFile(feedConf.Image)
		if err != nil {
			log.Printf("[WARN] failed to read image file %s, %v", feedConf.Image, err)
			w.WriteHeader(400)
			fmt.Fprintf(w, "error: can't read %s", feedConf.Image)
			return
		}
		log.Printf("[DEBUG] %s - %v - %s - %s - %s", r.URL.Path, time.Since(st), r.RemoteAddr, r.Referer(), r.UserAgent())
		w.Header().Set("Content-Type", "image/png")
		if _, err := w.Write(b); err != nil {
			log.Printf("[WARN] failed to send image, %s", err)
		}
	})

	// GET /list
	http.HandleFunc("/list", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "buckets: %+v", db.Buckets())
	})

	log.Print("[INFO] start http server on 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Printf("[ERROR] failed to start server, %v", err)
	}
}

func loadConfig(fname string) (res *proc.Conf, err error) {
	res = &proc.Conf{}
	data, err := ioutil.ReadFile(fname)
	if err != nil {
		return nil, err
	}

	if err = yaml.Unmarshal(data, res); err != nil {
		return nil, err
	}

	return res, nil
}

func setupLog(dbg bool) {
	filter := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DEBUG", "INFO", "WARN", "ERROR"},
		MinLevel: logutils.LogLevel("INFO"),
		Writer:   os.Stdout,
	}

	log.SetFlags(log.Ldate | log.Ltime)

	if dbg {
		log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
		filter.MinLevel = logutils.LogLevel("DEBUG")
	}
	log.SetOutput(filter)
}
