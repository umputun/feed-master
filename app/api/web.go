package api

import (
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/rest"

	"github.com/umputun/feed-master/app/feed"
)

var templates = template.Must(template.ParseGlob("webapp/templates/*"))

// GET /feed/{name} - renders page with list of items
func (s *Server) getFeedPageCtrl(w http.ResponseWriter, r *http.Request) {
	feedName := chi.URLParam(r, "name")
	items, err := s.Store.Load(feedName, s.Conf.System.MaxTotal)

	tmplData := struct {
		Items       []feed.Item
		Name        string
		Description string
		Link        string
		LastUpdate  time.Time
		Version     string
	}{
		Items:       items,
		Name:        s.Conf.Feeds[feedName].Title,
		Description: s.Conf.Feeds[feedName].Description,
		Link:        s.Conf.Feeds[feedName].Link,
		LastUpdate:  items[0].DT,
		Version:     s.Version,
	}

	if err != nil {
		rest.SendErrorJSON(w, r, log.Default(), http.StatusBadRequest, err, "failed to get feed")
		return
	}

	if err := templates.ExecuteTemplate(w, "feed.tmpl", &tmplData); err != nil {
		s.renderErrorPage(w, r, err, http.StatusInternalServerError)
		return
	}

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

// serves static files from /web
func (s *Server) addFileServer(r chi.Router, path string, root http.FileSystem) {
	fs := http.StripPrefix(path, http.FileServer(root))

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// don't show dirs, just serve files
		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		fs.ServeHTTP(w, r)
	}))
}
