// Package api provides rest-like server
package api

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth_chi"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/go-pkgz/lcw"
	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/rest"
	"github.com/go-pkgz/rest/logger"

	"github.com/umputun/feed-master/app/feed"
	"github.com/umputun/feed-master/app/proc"
)

// Server provides HTTP API
type Server struct {
	Version string
	Conf    proc.Conf
	Store   *proc.BoltDB

	httpServer *http.Server
	cache      lcw.LoadingCache
}

// Run starts http server for API with all routes
func (s *Server) Run(port int) {
	var err error
	if s.cache, err = lcw.NewExpirableCache(lcw.TTL(time.Minute*5), lcw.MaxCacheSize(10*1024*1024)); err != nil {
		log.Printf("[PANIC] failed to make loading cache, %v", err)
		return
	}

	router := chi.NewRouter()
	router.Use(middleware.RealIP, rest.Recoverer(log.Default()))
	router.Use(middleware.Throttle(1000), middleware.Timeout(60*time.Second))
	router.Use(rest.AppInfo("feed-master", "umputun", s.Version), rest.Ping)
	router.Use(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(5, nil)))

	s.httpServer = &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	router.Group(func(rimg chi.Router) {
		l := logger.New(logger.Log(log.Default()), logger.Prefix("[DEBUG]"))
		rimg.Use(l.Handler)
		rimg.Get("/images/{name}", s.getImageCtrl)
		rimg.Get("/image/{name}", s.getImageCtrl)
		rimg.Head("/image/{name}", s.getImageHeadCtrl)
		rimg.Head("/images/{name}", s.getImageHeadCtrl)
	})

	router.Group(func(rrss chi.Router) {
		l := logger.New(logger.Log(log.Default()), logger.Prefix("[INFO]"))
		rrss.Use(l.Handler)
		rrss.Get("/rss/{name}", s.getFeedCtrl)
		rrss.Get("/list", s.getListCtrl)
		rrss.Get("/feed/{name}", s.getFeedPageCtrl)
	})

	s.addFileServer(router, "/static", http.Dir(filepath.Join("webapp", "static")))
	err = s.httpServer.ListenAndServe()
	log.Printf("[WARN] http server terminated, %s", err)
}

// GET /rss/{name} - returns rss for given feeds set
func (s *Server) getFeedCtrl(w http.ResponseWriter, r *http.Request) {
	feedName := chi.URLParam(r, "name")
	items, err := s.Store.Load(feedName, s.Conf.System.MaxTotal)
	if err != nil {
		rest.SendErrorJSON(w, r, log.Default(), http.StatusBadRequest, err, "failed to get feed")
		return
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
	}

	// replace link to UI page
	if s.Conf.System.BaseURL != "" {
		rss.Link = s.Conf.System.BaseURL + "/feed/" + feedName
	}

	b, err := xml.MarshalIndent(&rss, "", "  ")
	if err != nil {
		rest.SendErrorJSON(w, r, log.Default(), http.StatusInternalServerError, err, "failed to marshal rss")
		return
	}
	w.Header().Set("Content-Type", "application/xml; charset=UTF-8")
	_, _ = fmt.Fprintf(w, "%s", string(b))
}

// GET /image/{name}
func (s *Server) getImageCtrl(w http.ResponseWriter, r *http.Request) {
	fm := chi.URLParam(r, "name")
	fm = strings.TrimRight(fm, ".png")
	feedConf, found := s.Conf.Feeds[fm]
	if !found {
		rest.SendErrorJSON(w, r, log.Default(), http.StatusBadRequest,
			errors.New("image "+chi.URLParam(r, "name")+" not found"), "failed to load image")
		return
	}

	b, err := ioutil.ReadFile(feedConf.Image)
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

// HEAD /image/{name}
func (s *Server) getImageHeadCtrl(w http.ResponseWriter, r *http.Request) {
	fm := chi.URLParam(r, "name")
	fm = strings.TrimRight(fm, ".png")
	feedConf, found := s.Conf.Feeds[fm]
	if !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	info, err := os.Stat(feedConf.Image)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", strconv.Itoa(int(info.Size())))
	w.WriteHeader(http.StatusOK)
}

// GET /list - returns feed's image
func (s *Server) getListCtrl(w http.ResponseWriter, r *http.Request) {
	buckets, err := s.Store.Buckets()
	if err != nil {
		rest.SendErrorJSON(w, r, log.Default(), http.StatusInternalServerError, err, "failed to read list")
		return
	}
	render.JSON(w, r, buckets)
}
