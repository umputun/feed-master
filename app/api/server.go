// Package api provides rest-like server
package api

import (
	"context"
	"crypto/subtle"
	"encoding/xml"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/didip/tollbooth/v6"
	"github.com/didip/tollbooth_chi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-pkgz/lcw"
	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/rest"
	"github.com/go-pkgz/rest/logger"

	"github.com/umputun/feed-master/app/config"
	"github.com/umputun/feed-master/app/feed"
	"github.com/umputun/feed-master/app/youtube"
	ytfeed "github.com/umputun/feed-master/app/youtube/feed"
)

//go:generate moq -out mocks/yt_service.go -pkg mocks -skip-ensure -fmt goimports . YoutubeSvc
//go:generate moq -out mocks/store.go -pkg mocks -skip-ensure -fmt goimports . Store

// Server provides HTTP API
type Server struct {
	Version       string
	Conf          config.Conf
	Store         Store
	YoutubeSvc    YoutubeSvc
	TemplLocation string
	AdminPasswd   string

	httpServer *http.Server
	cache      lcw.LoadingCache
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
	Load(fmFeed string, max int, skipJunk bool) ([]feed.Item, error)
}

// Run starts http server for API with all routes
func (s *Server) Run(ctx context.Context, port int) {
	log.Printf("[INFO] starting server on port %d", port)
	var err error
	if s.cache, err = lcw.NewExpirableCache(lcw.TTL(time.Minute*5), lcw.MaxCacheSize(10*1024*1024)); err != nil {
		log.Printf("[PANIC] failed to make loading cache, %v", err)
		return
	}

	go func() {
		<-ctx.Done()
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
	s.templates = template.Must(template.ParseGlob(s.TemplLocation))

	s.httpServer = &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           s.router(),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	err = s.httpServer.ListenAndServe()
	log.Printf("[WARN] http server terminated, %s", err)
}

func (s *Server) router() *chi.Mux {
	router := chi.NewRouter()
	router.Use(middleware.RealIP, rest.Recoverer(log.Default()), middleware.GetHead)
	router.Use(middleware.Throttle(1000), middleware.Timeout(60*time.Second))
	router.Use(rest.AppInfo("feed-master", "umputun", s.Version), rest.Ping)
	router.Use(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(5, nil)))
	router.Group(func(rimg chi.Router) {
		l := logger.New(logger.Log(log.Default()), logger.Prefix("[DEBUG]"), logger.IPfn(logger.AnonymizeIP))
		rimg.Use(l.Handler)
		rimg.Get("/images/{name}", s.getImageCtrl)
		rimg.Get("/image/{name}", s.getImageCtrl)
	})

	router.Group(func(rrss chi.Router) {
		l := logger.New(logger.Log(log.Default()), logger.Prefix("[INFO]"), logger.IPfn(logger.AnonymizeIP))
		rrss.Use(l.Handler)
		rrss.Get("/rss/{name}", s.getFeedCtrl)
		rrss.Head("/rss/{name}", s.getFeedCtrl)
		rrss.Get("/list", s.getListCtrl)
		rrss.Get("/feed/{name}", s.getFeedPageCtrl)
		rrss.Get("/feed/{name}/sources", s.getSourcesPageCtrl)
		rrss.Get("/feeds", s.getFeedsPageCtrl)
	})

	router.Route("/yt", func(r chi.Router) {

		auth := rest.BasicAuth(func(user, passwd string) bool {
			return (subtle.ConstantTimeCompare([]byte(s.AdminPasswd), []byte(passwd)) +
				subtle.ConstantTimeCompare([]byte("admin"), []byte(user))) == 2
		})

		l := logger.New(logger.Log(log.Default()), logger.Prefix("[INFO]"), logger.IPfn(logger.AnonymizeIP))
		r.Use(l.Handler)
		r.Get("/rss/{channel}", s.getYoutubeFeedCtrl)
		r.With(auth).Post("/rss/generate", s.regenerateRSSCtrl)
		r.With(auth).Delete("/entry/{channel}/{video}", s.removeEntryCtrl)
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
			router.Mount(baseYtURL.Path, ytfs)
		} else {
			log.Printf("[WARN] can't start static file server for yt, %v", fsErr)
		}
	}

	fs, err := rest.NewFileServer("/static", filepath.Join("webapp", "static"))
	if err == nil {
		router.Mount("/static", fs)
	} else {
		log.Printf("[WARN] can't start static file server, %v", err)
	}
	return router
}

// GET /rss/{name} - returns rss for given feeds set
func (s *Server) getFeedCtrl(w http.ResponseWriter, r *http.Request) {
	feedName := chi.URLParam(r, "name")
	items, err := s.Store.Load(feedName, s.Conf.System.MaxTotal, true)
	if err != nil {
		rest.SendErrorJSON(w, r, log.Default(), http.StatusBadRequest, err, "failed to get feed")
		return
	}

	for i, itm := range items {
		// add ts suffix to titles
		switch s.Conf.Feeds[feedName].ExtendDateTitle {
		case "yyyyddmm":
			items[i].Title = fmt.Sprintf("%s (%s)", itm.Title, itm.DT.Format("2006-02-01"))
		case "yyyymmdd":
			items[i].Title = fmt.Sprintf("%s (%s)", itm.Title, itm.DT.Format("2006-01-02"))
		}
	}

	rss := feed.Rss2{
		Version:       "2.0",
		ItemList:      items,
		Title:         s.Conf.Feeds[feedName].Title,
		Description:   s.Conf.Feeds[feedName].Description,
		Language:      s.Conf.Feeds[feedName].Language,
		Link:          s.Conf.Feeds[feedName].Link,
		PubDate:       items[0].PubDate,
		LastBuildDate: time.Now().Format(time.RFC822Z),
		NsItunes:      "http://www.itunes.com/dtds/podcast-1.0.dtd",
		NsMedia:       "http://search.yahoo.com/mrss/",
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
		return
	}

	w.Header().Set("Content-Type", "application/xml; charset=UTF-8")
	res := `<?xml version="1.0" encoding="UTF-8"?>` + "\n" + string(b)
	// this hack to avoid having different items for marshal and unmarshal due to "itunes" namespace
	res = strings.Replace(res, "<duration>", "<itunes:duration>", -1)
	res = strings.Replace(res, "</duration>", "</itunes:duration>", -1)

	_, _ = fmt.Fprintf(w, "%s", res)
}

// GET /image/{name}
func (s *Server) getImageCtrl(w http.ResponseWriter, r *http.Request) {
	fm := chi.URLParam(r, "name")
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
			errors.New("can't read  "+chi.URLParam(r, "name")), "failed to read image")
		return
	}
	w.Header().Set("Content-Type", "image/png")
	if _, err := w.Write(b); err != nil {
		log.Printf("[WARN] failed to send image, %s", err)
	}
}

// GET /list - returns feed's image
func (s *Server) getListCtrl(w http.ResponseWriter, r *http.Request) {
	feeds := s.feeds()
	render.JSON(w, r, feeds)
}

// GET /yt/rss/{channel} - returns rss for given youtube channel
func (s *Server) getYoutubeFeedCtrl(w http.ResponseWriter, r *http.Request) {
	channel := chi.URLParam(r, "channel")

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
	err := s.YoutubeSvc.RemoveEntry(ytfeed.Entry{ChannelID: chi.URLParam(r, "channel"), VideoID: chi.URLParam(r, "video")})
	if err != nil {
		rest.SendErrorJSON(w, r, log.Default(), http.StatusInternalServerError, err, "failed to remove entry")
		return
	}
	rest.RenderJSON(w, rest.JSON{"status": "ok", "removed": chi.URLParam(r, "video")})
}

func (s *Server) feeds() []string {
	feeds := make([]string, 0, len(s.Conf.Feeds))
	for k := range s.Conf.Feeds {
		feeds = append(feeds, k)
	}
	return feeds
}
