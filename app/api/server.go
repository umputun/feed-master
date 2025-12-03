// Package api provides rest-like server
package api

import (
	"context"
	"crypto/subtle"
	"encoding/xml"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-pkgz/lcw/v2"
	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/rest"
	"github.com/go-pkgz/rest/logger"
	"github.com/go-pkgz/routegroup"

	"github.com/umputun/feed-master/app/config"
	"github.com/umputun/feed-master/app/feed"
	"github.com/umputun/feed-master/app/youtube"
	ytfeed "github.com/umputun/feed-master/app/youtube/feed"
)

//go:generate moq -out mocks/yt_service.go -pkg mocks -skip-ensure -fmt goimports . YoutubeSvc
//go:generate moq -out mocks/store.go -pkg mocks -skip-ensure -fmt goimports . Store
//go:generate moq -out mocks/youtube_store.go -pkg mocks -skip-ensure -fmt goimports . YoutubeStore

// Server provides HTTP API
type Server struct {
	Version       string
	Conf          config.Conf
	Store         Store
	YoutubeStore  YoutubeStore
	YoutubeSvc    YoutubeSvc
	TemplLocation string
	AdminPasswd   string

	httpServer *http.Server
	cache      lcw.LoadingCache[[]byte]
	templates  *template.Template
}

// YoutubeSvc provides access to youtube's audio rss
type YoutubeSvc interface {
	RSSFeed(cinfo youtube.FeedInfo) (string, error)
	StoreRSS(chanID, rss string) error
	RemoveEntry(entry ytfeed.Entry) error
}

// Store provides access to feed data
type Store interface {
	Load(fmFeed string, maxItems int, skipJunk bool) ([]feed.Item, error)
}

// YoutubeStore provides access to YouTube channel data
type YoutubeStore interface {
	Load(channelID string, maxItems int) ([]ytfeed.Entry, error)
}

// Run starts http server for API with all routes
func (s *Server) Run(ctx context.Context, port int) {
	log.Printf("[INFO] starting server on port %d", port)
	var err error
	o := lcw.NewOpts[[]byte]()
	if s.cache, err = lcw.NewExpirableCache(o.TTL(time.Minute*3), o.MaxCacheSize(10*1024*1024)); err != nil {
		log.Printf("[PANIC] failed to make loading cache, %v", err)
		return
	}

	serverLock := sync.Mutex{}
	go func() {
		<-ctx.Done()
		serverLock.Lock()
		defer serverLock.Unlock()
		if s.httpServer != nil {
			if clsErr := s.httpServer.Close(); clsErr != nil {
				log.Printf("[ERROR] failed to close proxy http server, %v", clsErr)
			}
		}
	}()

	if s.TemplLocation == "" {
		s.TemplLocation = "webapp/templates/*"
	}
	log.Printf("[DEBUG] loading templates from %s", s.TemplLocation)
	s.loadTemplates()

	serverLock.Lock()
	s.httpServer = &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           s.router(),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      s.Conf.System.HTTPResponseTimeout,
		IdleTimeout:       30 * time.Second,
	}
	serverLock.Unlock()
	err = s.httpServer.ListenAndServe()
	log.Printf("[WARN] http server terminated, %s", err)
}

// loadTemplates loads templates with custom functions
func (s *Server) loadTemplates() {
	funcMap := template.FuncMap{
		"currentYear": func() int {
			return time.Now().Year()
		},
	}
	s.templates = template.Must(template.New("").Funcs(funcMap).ParseGlob(s.TemplLocation))
}

func (s *Server) router() http.Handler {
	router := routegroup.New(http.NewServeMux())
	router.Use(rest.RealIP, rest.Recoverer(log.Default()))
	router.Use(rest.Throttle(1000), timeout(60*time.Second))
	router.Use(rest.AppInfo("feed-master", "umputun", s.Version), rest.Ping)
	router.Use(rest.Throttle(5)) // rate limiter, replaces tollbooth

	router.Group().Route(func(rimg *routegroup.Bundle) {
		l := logger.New(logger.Log(log.Default()), logger.Prefix("[DEBUG]"), logger.IPfn(logger.AnonymizeIP))
		rimg.Use(l.Handler)
		rimg.HandleFunc("GET /images/{name}", s.getImageCtrl)
		rimg.HandleFunc("GET /image/{name...}", s.getImageCtrl) // handles both /image/foo and /image/foo.png
	})

	router.Group().Route(func(rrss *routegroup.Bundle) {
		l := logger.New(logger.Log(log.Default()), logger.Prefix("[INFO]"), logger.IPfn(logger.AnonymizeIP))
		rrss.Use(l.Handler)
		rrss.HandleFunc("GET /rss/{name}", s.getFeedCtrl)
		rrss.HandleFunc("GET /list", s.getListCtrl)
		rrss.HandleFunc("GET /feed/{name}", s.getFeedPageCtrl)
		rrss.HandleFunc("GET /feed/{name}/sources", s.getSourcesPageCtrl)
		rrss.HandleFunc("GET /feed/{name}/source/{source}", s.getFeedSourceCtrl)
		rrss.HandleFunc("GET /feeds", s.getFeedsPageCtrl)
	})

	router.HandleFunc("GET /config", func(w http.ResponseWriter, _ *http.Request) { rest.RenderJSON(w, s.Conf) })

	router.Mount("/yt").Route(func(r *routegroup.Bundle) {
		auth := rest.BasicAuth(func(user, passwd string) bool {
			return (subtle.ConstantTimeCompare([]byte(s.AdminPasswd), []byte(passwd)) +
				subtle.ConstantTimeCompare([]byte("admin"), []byte(user))) == 2
		})

		l := logger.New(logger.Log(log.Default()), logger.Prefix("[INFO]"), logger.IPfn(logger.AnonymizeIP))
		r.Use(l.Handler)
		r.HandleFunc("GET /rss/{channel}", s.getYoutubeFeedCtrl)
		r.HandleFunc("GET /channels", s.getYoutubeChannelsPageCtrl)
		r.With(auth).HandleFunc("POST /rss/generate", s.regenerateRSSCtrl)
		r.With(auth).HandleFunc("DELETE /entry/{channel}/{video}", s.removeEntryCtrl)
	})

	if s.Conf.YouTube.BaseURL != "" {
		baseYtURL, parseErr := url.Parse(s.Conf.YouTube.BaseURL)
		if parseErr != nil {
			log.Printf("[ERROR] failed to parse base url %s, %v", s.Conf.YouTube.BaseURL, parseErr)
		}

		if mkdirErr := os.MkdirAll(s.Conf.YouTube.FilesLocation, 0o750); mkdirErr != nil {
			log.Printf("[ERROR] failed to create directory %s, %v", s.Conf.YouTube.FilesLocation, mkdirErr)
		}

		ytfs, fsErr := rest.NewFileServer(baseYtURL.Path, s.Conf.YouTube.FilesLocation)
		if fsErr == nil {
			router.Handle(baseYtURL.Path+"/{file...}", ytfs)
		} else {
			log.Printf("[WARN] can't start static file server for yt, %v", fsErr)
		}
	}

	fs, err := rest.NewFileServer("/static", filepath.Join("webapp", "static"))
	if err == nil {
		router.Handle("/static/{file...}", fs)
	} else {
		log.Printf("[WARN] can't start static file server, %v", err)
	}
	return router
}

// GET /rss/{name} - returns rss for given feeds set
func (s *Server) getFeedCtrl(w http.ResponseWriter, r *http.Request) {
	feedName := r.PathValue("name")

	data, err := s.cache.Get("feed::"+feedName, func() ([]byte, error) {
		items, err := s.Store.Load(feedName, s.Conf.System.MaxTotal, true)
		if err != nil {
			return nil, err
		}

		for i, itm := range items {
			// add ts suffix to titles
			switch s.Conf.Feeds[feedName].ExtendDateTitle {
			case "yyyyddmm":
				items[i].Title = fmt.Sprintf("%s (%s)", itm.Title, itm.DT.Format("2006-02-01")) // nolint
			case "yyyymmdd":
				items[i].Title = fmt.Sprintf("%s (%s)", itm.Title, itm.DT.Format("2006-01-02"))
			}
		}

		rss := feed.Rss2{
			Version:        "2.0",
			ItemList:       items,
			Title:          s.Conf.Feeds[feedName].Title,
			Description:    s.Conf.Feeds[feedName].Description,
			Language:       s.Conf.Feeds[feedName].Language,
			Link:           s.Conf.Feeds[feedName].Link,
			PubDate:        items[0].PubDate,
			LastBuildDate:  time.Now().Format(time.RFC822Z),
			ItunesAuthor:   s.Conf.Feeds[feedName].Author,
			ItunesExplicit: "no",
			ItunesOwner: &feed.ItunesOwner{
				Name:  "Feed Master",
				Email: s.Conf.Feeds[feedName].OwnerEmail,
			},
			NsItunes: "http://www.itunes.com/dtds/podcast-1.0.dtd",
			NsMedia:  "http://search.yahoo.com/mrss/",
		}

		// replace link to UI page
		if s.Conf.System.BaseURL != "" {
			baseURL := strings.TrimSuffix(s.Conf.System.BaseURL, "/")
			rss.Link = baseURL + "/feed/" + feedName
			imagesURL := baseURL + "/images/" + feedName
			rss.ItunesImage = &feed.ItunesImg{URL: imagesURL}
			rss.MediaThumbnail = &feed.MediaThumbnail{URL: imagesURL}
		}

		b, err := xml.MarshalIndent(&rss, "", "  ")
		if err != nil {
			rest.SendErrorJSON(w, r, log.Default(), http.StatusInternalServerError, err, "failed to marshal rss")
			return nil, fmt.Errorf("failed to marshal rss for %s: %w", feedName, err)
		}

		res := `<?xml version="1.0" encoding="UTF-8"?>` + "\n" + string(b)

		// this hack to avoid having different items for marshal and unmarshal due to "itunes" namespace
		res = strings.ReplaceAll(res, "<duration>", "<itunes:duration>")
		res = strings.ReplaceAll(res, "</duration>", "</itunes:duration>")

		return []byte(res), nil
	})

	if err != nil {
		rest.SendErrorJSON(w, r, log.Default(), http.StatusBadRequest, err, "failed to get feed")
		return
	}

	w.Header().Set("Content-Type", "application/xml; charset=UTF-8")
	_, _ = fmt.Fprintf(w, "%s", data)
}

// GET /image/{name}
func (s *Server) getImageCtrl(w http.ResponseWriter, r *http.Request) {
	fm := r.PathValue("name")
	fm = strings.TrimSuffix(fm, ".png")
	feedConf, found := s.Conf.Feeds[fm]
	if !found {
		rest.SendErrorJSON(w, r, log.Default(), http.StatusBadRequest,
			fmt.Errorf("image %s not found", fm), "failed to load image")
		return
	}

	b, err := os.ReadFile(feedConf.Image)
	if err != nil {
		rest.SendErrorJSON(w, r, log.Default(), http.StatusBadRequest,
			fmt.Errorf("can't read %s", r.PathValue("name")), "failed to read image")
		return
	}
	w.Header().Set("Content-Type", "image/png")
	if _, err := w.Write(b); err != nil {
		log.Printf("[WARN] failed to send image, %s", err)
	}
}

// GET /list - returns list of feeds
func (s *Server) getListCtrl(w http.ResponseWriter, _ *http.Request) {
	feeds := s.feeds()
	rest.RenderJSON(w, feeds)
}

// GET /yt/rss/{channel} - returns rss for given youtube channel
func (s *Server) getYoutubeFeedCtrl(w http.ResponseWriter, r *http.Request) {
	channel := r.PathValue("channel")

	fi := youtube.FeedInfo{ID: channel}
	for _, f := range s.Conf.YouTube.Channels {
		if f.ID == channel {
			fi = f
			break
		}
	}

	res, err := s.YoutubeSvc.RSSFeed(fi)
	if err != nil {
		rest.SendErrorJSON(w, r, log.Default(), http.StatusInternalServerError, err, "failed to read yt list")
		return
	}

	w.Header().Set("Content-Type", "application/xml; charset=UTF-8")
	res = `<?xml version="1.0" encoding="UTF-8"?>` + "\n" + res
	_, _ = fmt.Fprintf(w, "%s", res)
}

// POST /yt/rss/generate - generates rss for all (each) youtube channels
func (s *Server) regenerateRSSCtrl(w http.ResponseWriter, r *http.Request) {

	for _, f := range s.Conf.YouTube.Channels {
		res, err := s.YoutubeSvc.RSSFeed(youtube.FeedInfo{ID: f.ID})
		if err != nil {
			rest.SendErrorJSON(w, r, log.Default(), http.StatusInternalServerError, err, "failed to read yt rss for "+f.ID)
			return
		}
		if err := s.YoutubeSvc.StoreRSS(f.ID, res); err != nil {
			rest.SendErrorJSON(w, r, log.Default(), http.StatusInternalServerError, err, "failed to store yt rss for "+f.ID)
			return
		}
	}
	rest.RenderJSON(w, rest.JSON{"status": "ok", "feeds": len(s.Conf.YouTube.Channels)})
}

// DELETE /yt/entry/{channel}/{video} - deletes entry from youtube channel and videID
func (s *Server) removeEntryCtrl(w http.ResponseWriter, r *http.Request) {
	err := s.YoutubeSvc.RemoveEntry(ytfeed.Entry{ChannelID: r.PathValue("channel"), VideoID: r.PathValue("video")})
	if err != nil {
		rest.SendErrorJSON(w, r, log.Default(), http.StatusInternalServerError, err, "failed to remove entry")
		return
	}
	rest.RenderJSON(w, rest.JSON{"status": "ok", "removed": r.PathValue("video")})
}

func (s *Server) feeds() []string {
	feeds := make([]string, 0, len(s.Conf.Feeds))
	for k := range s.Conf.Feeds {
		feeds = append(feeds, k)
	}
	return feeds
}

// timeout wraps http.TimeoutHandler as middleware
func timeout(dt time.Duration) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.TimeoutHandler(h, dt, "timeout")
	}
}
