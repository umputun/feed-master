package api

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/go-pkgz/rest"

	"github.com/umputun/feed-master/app/config"
	"github.com/umputun/feed-master/app/feed"
	"github.com/umputun/feed-master/app/youtube"
	ytfeed "github.com/umputun/feed-master/app/youtube/feed"
)

// GET /feed/{name} - renders page with list of items
func (s *Server) getFeedPageCtrl(w http.ResponseWriter, r *http.Request) {
	feedName := r.PathValue("name")

	data, err := s.cache.Get(feedName, func() ([]byte, error) {
		items, err := s.Store.Load(feedName, s.Conf.System.MaxTotal, false)
		if err != nil {
			return nil, err
		}

		// fill formatted duration
		for i, item := range items {
			if item.Duration == "" {
				continue
			}
			d, e := time.ParseDuration(item.Duration + "s")
			if e != nil {
				continue
			}
			items[i].DurationFmt = d.String()
		}

		tmplData := struct {
			Items           []feed.Item
			Name            string
			Description     string
			Link            string
			LastUpdate      time.Time
			SinceLastUpdate string
			Feeds           int
			Version         string
			RSSLink         string
			SourcesLink     string
			TelegramChannel string
		}{
			Items:           items,
			Name:            s.Conf.Feeds[feedName].Title,
			Description:     s.Conf.Feeds[feedName].Description,
			Link:            s.Conf.Feeds[feedName].Link,
			LastUpdate:      items[0].DT.In(time.UTC),
			SinceLastUpdate: humanize.Time(items[0].DT),
			Feeds:           len(s.Conf.Feeds[feedName].Sources),
			Version:         s.Version,
			RSSLink:         s.Conf.System.BaseURL + "/rss/" + feedName,
			SourcesLink:     s.Conf.System.BaseURL + "/feed/" + feedName + "/sources",
			TelegramChannel: s.Conf.Feeds[feedName].TelegramChannel,
		}

		res := bytes.NewBuffer(nil)
		err = s.templates.ExecuteTemplate(res, "feed.tmpl", &tmplData)
		return res.Bytes(), err
	})

	if err != nil {
		s.renderErrorPage(w, r, err, 400)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data) // nolint
}

// GET /feed/{name}/source/{source} - renders feed's source page with list of items
func (s *Server) getFeedSourceCtrl(w http.ResponseWriter, r *http.Request) {
	feedName := r.PathValue("name")
	sourceNameRaw := r.PathValue("source")
	var err error

	sourceName, err := url.QueryUnescape(sourceNameRaw)
	if err != nil {
		s.renderErrorPage(w, r, err, 400)
		return
	}

	data, err := s.cache.Get(feedName+sourceName, func() ([]byte, error) {
		if _, ok := s.Conf.Feeds[feedName]; !ok {
			return nil, fmt.Errorf("feed %s not found", feedName)
		}

		var feedInfo youtube.FeedInfo
		for _, k := range s.Conf.YouTube.Channels {
			if k.Name == sourceName {
				feedInfo = k
				break
			}
		}
		if feedInfo.ID == "" {
			return nil, fmt.Errorf("feed %s does not have source %s", feedName, sourceName)
		}

		items, er := s.YoutubeStore.Load(feedInfo.ID, s.Conf.YouTube.MaxItems)
		if er != nil {
			return nil, er
		}

		// fill formatted duration and file path
		for i, item := range items {
			if item.Duration == 0 {
				continue
			}
			d := time.Duration(int(time.Second) * item.Duration)
			items[i].DurationFmt = d.String()
			items[i].File = s.Conf.YouTube.BaseURL + "/" + path.Base(item.File)
		}

		tmplData := struct {
			Items           []ytfeed.Entry
			Name            string
			Link            string
			LastUpdate      time.Time
			SinceLastUpdate string
			Feeds           int
			Version         string
			RSSLink         string
		}{
			Items:           items,
			Name:            feedInfo.Name,
			Link:            "https://youtube.com/channel/" + feedInfo.ID,
			LastUpdate:      items[0].Published.In(time.UTC),
			SinceLastUpdate: humanize.Time(items[0].Published),
			Feeds:           len(items),
			Version:         s.Version,
			RSSLink:         s.Conf.System.BaseURL + "/yt/rss/" + feedInfo.ID,
		}
		if feedInfo.Type == ytfeed.FTPlaylist {
			tmplData.Link = "https://www.youtube.com/playlist?list=" + feedInfo.ID
		}

		res := bytes.NewBuffer(nil)
		err = s.templates.ExecuteTemplate(res, "source.tmpl", &tmplData)
		return res.Bytes(), err
	})

	if err != nil {
		s.renderErrorPage(w, r, err, 400)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data) // nolint
}

// GET /feeds - renders page with list of feeds
func (s *Server) getFeedsPageCtrl(w http.ResponseWriter, r *http.Request) {
	data, err := s.cache.Get("feeds", func() ([]byte, error) {

		feeds := s.feeds()

		type feedItem struct {
			config.Feed
			FeedURL     string
			Sources     int
			SourcesLink string
			LastUpdated time.Time
		}
		var feedItems []feedItem
		for _, f := range feeds {
			items, loadErr := s.Store.Load(f, s.Conf.System.MaxTotal, true)
			if loadErr != nil {
				continue
			}
			feedConf := s.Conf.Feeds[f]
			item := feedItem{
				Feed:        feedConf,
				FeedURL:     s.Conf.System.BaseURL + "/feed/" + f,
				Sources:     len(feedConf.Sources),
				SourcesLink: s.Conf.System.BaseURL + "/feed/" + f + "/sources",
				LastUpdated: items[0].DT.In(time.UTC),
			}
			feedItems = append(feedItems, item)
		}

		tmplData := struct {
			Feeds      []feedItem
			FeedsCount int
		}{
			Feeds:      feedItems,
			FeedsCount: len(feedItems),
		}

		res := bytes.NewBuffer(nil)
		err := s.templates.ExecuteTemplate(res, "feeds.tmpl", &tmplData)
		return res.Bytes(), err
	})

	if err != nil {
		s.renderErrorPage(w, r, err, 400)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data) // nolint
}

// GET /yt/channels - renders page with list of YouTube channels
func (s *Server) getYoutubeChannelsPageCtrl(w http.ResponseWriter, r *http.Request) {
	data, err := s.cache.Get("channels", func() ([]byte, error) {
		type channelItem struct {
			youtube.FeedInfo
			ChannelURL  string
			LastUpdated time.Time
			RssURL      string
		}
		var channelItems []channelItem

		for _, k := range s.Conf.YouTube.Channels {
			items, loadErr := s.YoutubeStore.Load(k.ID, 1)
			if loadErr != nil {
				continue
			}
			item := channelItem{
				FeedInfo:    k,
				RssURL:      s.Conf.YouTube.BaseChanURL + k.ID,
				ChannelURL:  "https://youtube.com/channel/" + k.ID,
				LastUpdated: items[0].Published.In(time.UTC),
			}
			if k.Type == ytfeed.FTPlaylist {
				item.RssURL = s.Conf.YouTube.BasePlaylistURL + k.ID
				item.ChannelURL = "https://www.youtube.com/playlist?list=" + k.ID
			}
			channelItems = append(channelItems, item)
		}

		tmplData := struct {
			Channels []channelItem
			Count    int
		}{
			Channels: channelItems,
			Count:    len(channelItems),
		}

		res := bytes.NewBuffer(nil)
		err := s.templates.ExecuteTemplate(res, "channels.tmpl", &tmplData)
		return res.Bytes(), err
	})

	if err != nil {
		s.renderErrorPage(w, r, err, 400)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data) // nolint
}

// GET /feed/{name}/sources - renders page with feed's list of sources
func (s *Server) getSourcesPageCtrl(w http.ResponseWriter, r *http.Request) {
	feedName := r.PathValue("name")
	data, err := s.cache.Get(feedName+"-sources", func() ([]byte, error) {
		if _, ok := s.Conf.Feeds[feedName]; !ok {
			return nil, fmt.Errorf("feed %s not found", feedName)
		}
		feedConf := s.Conf.Feeds[feedName]

		type Source struct {
			Name string
			URL  string
		}

		tmplData := struct {
			Sources  []Source
			SrcCount int
		}{}

		for _, source := range feedConf.Sources {
			src := Source{
				Name: source.Name,
				URL:  s.Conf.System.BaseURL + "/feed/" + feedName + "/source/" + source.Name,
			}
			tmplData.Sources = append(tmplData.Sources, src)
		}
		tmplData.SrcCount = len(tmplData.Sources)

		res := bytes.NewBuffer(nil)
		err := s.templates.ExecuteTemplate(res, "sources.tmpl", &tmplData)
		return res.Bytes(), err
	})

	if err != nil {
		s.renderErrorPage(w, r, err, 400)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data) // nolint
}

func (s *Server) renderErrorPage(w http.ResponseWriter, _ *http.Request, err error, errCode int) { // nolint
	tmplData := struct {
		Status int
		Error  string
	}{Status: errCode, Error: err.Error()}

	if err := s.templates.ExecuteTemplate(w, "error.tmpl", &tmplData); err != nil {
		_ = rest.EncodeJSON(w, http.StatusInternalServerError, rest.JSON{"error": err.Error()})
		return
	}
}
