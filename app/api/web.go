package api

import (
	"bytes"
	"github.com/umputun/feed-master/app/proc"
	"html/template"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/go-pkgz/rest"

	"github.com/umputun/feed-master/app/feed"
)

var templates = template.Must(template.ParseGlob("webapp/templates/*"))

// GET /feed/{name} - renders page with list of items
func (s *Server) getFeedPageCtrl(w http.ResponseWriter, r *http.Request) {
	feedName := chi.URLParam(r, "name")

	data, err := s.cache.Get(feedName, func() (interface{}, error) {
		items, err := s.Store.Load(feedName, s.Conf.System.MaxTotal, false)
		if err != nil {
			return nil, err
		}
		tmplData := struct {
			Items       []feed.Item
			Name        string
			Description string
			Link        string
			LastUpdate  time.Time
			Feeds       int
			Version     string
			RSSLink     string
		}{
			Items:       items,
			Name:        s.Conf.Feeds[feedName].Title,
			Description: s.Conf.Feeds[feedName].Description,
			Link:        s.Conf.Feeds[feedName].Link,
			LastUpdate:  items[0].DT,
			Feeds:       len(s.Conf.Feeds[feedName].Sources),
			Version:     s.Version,
			RSSLink:     s.Conf.System.BaseURL + "/rss/" + feedName,
		}

		res := bytes.NewBuffer(nil)
		err = templates.ExecuteTemplate(res, "feed.tmpl", &tmplData)
		return res.Bytes(), err
	})

	if err != nil {
		s.renderErrorPage(w, r, err, 400)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data.([]byte)) // nolint
}

// GET /feeds - renders page with list of feeds
func (s *Server) getFeedsPageCtrl(w http.ResponseWriter, r *http.Request) {
	data, err := s.cache.Get("feeds", func() (interface{}, error) {
		feeds, err := s.Store.Buckets()
		if err != nil {
			return nil, err
		}

		type feedItem struct {
			proc.Feed
			FeedURL string
		}
		var feedItems []feedItem
		for _, f := range feeds {
			feedItems = append(
				feedItems,
				feedItem{Feed: s.Conf.Feeds[f], FeedURL: s.Conf.System.BaseURL + "/feed/" + f},
			)
		}

		tmplData := struct {
			Feeds      []feedItem
			FeedsCount int
		}{
			Feeds:      feedItems,
			FeedsCount: len(feedItems),
		}

		res := bytes.NewBuffer(nil)
		err = templates.ExecuteTemplate(res, "feeds.tmpl", &tmplData)
		return res.Bytes(), err
	})

	if err != nil {
		s.renderErrorPage(w, r, err, 400)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data.([]byte)) // nolint
}

func (s *Server) renderErrorPage(w http.ResponseWriter, r *http.Request, err error, errCode int) {
	tmplData := struct {
		Status int
		Error  string
	}{Status: errCode, Error: err.Error()}

	if err := templates.ExecuteTemplate(w, "error.tmpl", &tmplData); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, rest.JSON{"error": err.Error()})
		return
	}
}
