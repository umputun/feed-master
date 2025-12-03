// Package youtube provides loading audio from video files for given youtube channels
package youtube

import (
	"context"
	"crypto/sha1"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/bogem/id3v2/v2"
	log "github.com/go-pkgz/lgr"
	"github.com/google/uuid"

	rssfeed "github.com/umputun/feed-master/app/feed"
	ytfeed "github.com/umputun/feed-master/app/youtube/feed"
)

//go:generate moq -out mocks/downloader.go -pkg mocks -skip-ensure -fmt goimports . DownloaderService
//go:generate moq -out mocks/channel.go -pkg mocks -skip-ensure -fmt goimports . ChannelService
//go:generate moq -out mocks/store.go -pkg mocks -skip-ensure -fmt goimports . StoreService
//go:generate moq -out mocks/duration.go -pkg mocks -skip-ensure -fmt goimports . DurationService

// Service loads audio from youtube channels
type Service struct {
	Feeds           []FeedInfo
	Downloader      DownloaderService
	ChannelService  ChannelService
	Store           StoreService
	CheckDuration   time.Duration
	RSSFileStore    RSSFileStore
	DurationService DurationService
	KeepPerChannel  int
	RootURL         string
	SkipShorts      time.Duration

	YtDlpUpdDuration time.Duration
	YtDlpUpdCommand  string
}

// FeedInfo contains channel or feed ID, readable name and other per-feed info
type FeedInfo struct {
	Name     string      `yaml:"name"`
	ID       string      `yaml:"id"`
	Type     ytfeed.Type `yaml:"type"`
	Keep     int         `yaml:"keep"`
	Language string      `yaml:"lang"`
	Filter   FeedFilter  `yaml:"filter"`
}

// FeedFilter contains filter criteria for the feed
type FeedFilter struct {
	Include string `yaml:"include"`
	Exclude string `yaml:"exclude"`
}

// DownloaderService is an interface for downloading audio from youtube
type DownloaderService interface {
	Get(ctx context.Context, id string, fname string) (file string, err error)
}

// ChannelService is an interface for getting channel entries, i.e. the list of videos
type ChannelService interface {
	Get(ctx context.Context, chanID string, feedType ytfeed.Type) ([]ytfeed.Entry, error)
}

// StoreService is an interface for storing and loading metadata about downloaded audio
type StoreService interface {
	Save(entry ytfeed.Entry) (bool, error)
	Load(channelID string, maX int) ([]ytfeed.Entry, error)
	Exist(entry ytfeed.Entry) (bool, error)
	RemoveOld(channelID string, keep int) ([]string, error)
	Remove(entry ytfeed.Entry) error
	SetProcessed(entry ytfeed.Entry) error
	ResetProcessed(entry ytfeed.Entry) error
	CheckProcessed(entry ytfeed.Entry) (found bool, ts time.Time, err error)
	CountProcessed() (count int)
}

// DurationService is an interface for getting duration of audio file
type DurationService interface {
	File(fname string) int
}

// Do is a blocking function that downloads audio from youtube channels and updates metadata
func (s *Service) Do(ctx context.Context) error {
	log.Printf("[INFO] starting youtube service")
	lastYtDlpUpdate := time.Now()
	if s.SkipShorts > 0 {
		log.Printf("[DEBUG] skip youtube episodes shorter than %v", s.SkipShorts)
	}
	for _, f := range s.Feeds {
		log.Printf("[INFO] youtube feed %+v", f)
	}

	tick := time.NewTicker(s.CheckDuration)
	defer tick.Stop()

	if err := s.procChannels(ctx); err != nil {
		return fmt.Errorf("failed to process channels: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
			if s.YtDlpUpdDuration > 0 && time.Since(lastYtDlpUpdate) > s.YtDlpUpdDuration && s.YtDlpUpdCommand != "" {
				// update yt-dlp binary once in a while
				lastYtDlpUpdate = time.Now()
				s.execYtdlpUpdate(ctx, s.YtDlpUpdCommand)
			}
			if err := s.procChannels(ctx); err != nil {
				return fmt.Errorf("failed to process channels: %w", err)
			}
		}
	}
}

// RSSFeed generates RSS feed for given channel
func (s *Service) RSSFeed(fi FeedInfo) (string, error) {
	entries, err := s.Store.Load(fi.ID, s.keep(fi))
	if err != nil {
		return "", fmt.Errorf("failed to get channel entries: %w", err)
	}

	if len(entries) == 0 {
		return "", nil
	}

	items := []rssfeed.Item{}
	for _, entry := range entries {

		fileURL := s.RootURL + "/" + path.Base(entry.File)

		var fileSize int
		if fileInfo, fiErr := os.Stat(entry.File); fiErr != nil {
			log.Printf("[WARN] failed to get file size for %s (%s %s): %v", entry.File, entry.VideoID, entry.Title, fiErr)
		} else {
			fileSize = int(fileInfo.Size())
		}

		duration := ""
		if entry.Duration > 0 {
			duration = fmt.Sprintf("%d", entry.Duration)
		}

		items = append(items, rssfeed.Item{
			Title:       entry.Title,
			Description: entry.Media.Description,
			Link:        entry.Link.Href,
			PubDate:     entry.Published.In(time.UTC).Format(time.RFC1123Z),
			GUID:        entry.ChannelID + "::" + entry.VideoID,
			Author:      entry.Author.Name,
			Enclosure: rssfeed.Enclosure{
				URL:    fileURL,
				Type:   "audio/mpeg",
				Length: fileSize,
			},
			Duration: duration,
			DT:       time.Now(),
		})
	}

	rss := rssfeed.Rss2{
		Version:        "2.0",
		NsItunes:       "http://www.itunes.com/dtds/podcast-1.0.dtd",
		NsMedia:        "http://search.yahoo.com/mrss/",
		ItemList:       items,
		Title:          fi.Name,
		Description:    "generated by feed-master",
		Link:           entries[0].Author.URI,
		PubDate:        items[0].PubDate,
		LastBuildDate:  time.Now().Format(time.RFC1123Z),
		Language:       fi.Language,
		ItunesAuthor:   entries[0].Author.Name,
		ItunesExplicit: "no",
	}

	// set image from channel as rss thumbnail
	// TODO: we may want to load it locally in case if youtube doesn't like such remote usage of images
	if image := entries[0].Media.Thumbnail.URL; image != "" {
		rss.ItunesImage = &rssfeed.ItunesImg{URL: image}
		rss.MediaThumbnail = &rssfeed.MediaThumbnail{URL: image}
	}

	if fi.Type == ytfeed.FTPlaylist {
		rss.Link = "https://www.youtube.com/playlist?list=" + fi.ID
	}

	b, err := xml.MarshalIndent(&rss, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal rss: %w", err)
	}

	res := string(b)
	// this hack to avoid having different items for marshal and unmarshal due to "itunes" namespace
	res = strings.ReplaceAll(res, "<duration>", "<itunes:duration>")
	res = strings.ReplaceAll(res, "</duration>", "</itunes:duration>")
	return res, nil
}

// procChannels processes all channels, downloads audio, updates metadata and stores RSS
func (s *Service) procChannels(ctx context.Context) error {

	var allStats stats

	for _, feedInfo := range s.Feeds {
		entries, err := s.ChannelService.Get(ctx, feedInfo.ID, feedInfo.Type)
		if err != nil {
			log.Printf("[WARN] failed to get channel entries for %s: %s", feedInfo.ID, err)
			continue
		}
		log.Printf("[INFO] got %d entries for %s, limit to %d", len(entries), feedInfo.Name, s.keep(feedInfo))
		changed, processed := false, 0
		for i, entry := range entries {

			// exit right away if context is done
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			allStats.entries++
			if processed >= s.keep(feedInfo) {
				break
			}
			isAllowed, err := s.isAllowed(entry, feedInfo)
			if err != nil {
				return fmt.Errorf("failed to check if entry %s is relevant: %w", entry.VideoID, err)
			}
			if !isAllowed {
				log.Printf("[DEBUG] skipping filtered %s", entry.String())
				allStats.ignored++
				continue
			}

			ok, err := s.isNew(entry, feedInfo)
			if err != nil {
				return fmt.Errorf("failed to check if entry %s exists: %w", entry.VideoID, err)
			}
			if !ok {
				allStats.skipped++
				processed++
				continue
			}

			// got new entry, but with very old timestamp. skip it if we have already reached max capacity
			// (this is to eliminate the initial load) and this entry is older than the oldest one we have.
			// also marks it as processed as we don't want to process it again
			oldestEntry := s.oldestEntry()
			if entry.Published.Before(oldestEntry.Published) && s.countAllEntries() >= s.totalEntriesToKeep() {
				allStats.ignored++
				log.Printf("[INFO] skipping entry %s as it is older than the oldest one we have %s",
					entry.String(), oldestEntry.String())
				if procErr := s.Store.SetProcessed(entry); procErr != nil {
					log.Printf("[WARN] failed to set processed status for %s: %v", entry.VideoID, procErr)
				}
				continue
			}

			log.Printf("[INFO] new entry [%d] %s, %s, %s, %s", i+1, entry.VideoID, entry.Title, feedInfo.Name, entry.String())

			file, downErr := s.Downloader.Get(ctx, entry.VideoID, s.makeFileName(entry))
			if downErr != nil {
				allStats.ignored++
				if downErr == ytfeed.ErrSkip { // downloader decided to skip this entry
					log.Printf("[INFO] skipping %s", entry.String())
					continue
				}
				log.Printf("[WARN] failed to download %s: %s", entry.VideoID, downErr)
				continue
			}

			if short, duration := s.isShort(file); short {
				allStats.ignored++
				log.Printf("[INFO] skip short file %s (%v): %s, %s", file, duration, entry.VideoID, entry.String())
				if procErr := s.Store.SetProcessed(entry); procErr != nil {
					log.Printf("[WARN] failed to set processed status for %s: %v", entry.VideoID, procErr)
				}
				if errRm := os.Remove(file); errRm != nil {
					log.Printf("[WARN] failed to remove short video's %s file %s: %v", entry.VideoID, file, errRm)
				}
				continue
			}

			// update metadata
			if tagsErr := s.updateMp3Tags(file, entry, feedInfo); tagsErr != nil {
				log.Printf("[WARN] failed to update metadata for %s: %s", entry.VideoID, tagsErr)
			}

			processed++

			fsize := 0
			if fi, err := os.Stat(file); err == nil {
				fsize = int(fi.Size())
			} else {
				log.Printf("[WARN] failed to get file size for %s: %v", file, err)
			}

			log.Printf("[INFO] downloaded %s (%s) to %s, size: %d, channel: %+v", entry.VideoID, entry.Title, file, fsize, feedInfo)

			entry = s.update(entry, file, feedInfo)

			ok, saveErr := s.Store.Save(entry)
			if saveErr != nil {
				return fmt.Errorf("failed to save entry %+v: %w", entry, saveErr)
			}
			if !ok {
				log.Printf("[WARN] attempt to save dup entry %+v", entry)
			}
			changed = true
			if procErr := s.Store.SetProcessed(entry); procErr != nil {
				log.Printf("[WARN] failed to set processed status for %s: %v", entry.VideoID, procErr)
			}
			allStats.added++
			log.Printf("[INFO] saved %s (%s) to %s, channel: %+v", entry.VideoID, entry.Title, file, feedInfo)
		}
		allStats.processed += processed

		if changed {
			removed := s.removeOld(feedInfo)
			allStats.removed += removed

			// save rss feed to fs if there are new entries
			rss, rssErr := s.RSSFeed(feedInfo)
			if rssErr != nil {
				log.Printf("[WARN] failed to generate rss for %s: %s", feedInfo.Name, rssErr)
			} else {
				if err := s.RSSFileStore.Save(feedInfo.ID, rss); err != nil {
					log.Printf("[WARN] failed to save rss for %s: %s", feedInfo.Name, err)
				}
			}
		}
	}

	log.Printf("[INFO] all channels processed - channels: %d, %s, lifetime: %d, feed size: %d",
		len(s.Feeds), allStats.String(), s.Store.CountProcessed(), s.countAllEntries())

	newestEntry := s.newestEntry()
	log.Printf("[INFO] last entry: %s", newestEntry.String())

	return nil
}

// StoreRSS saves RSS feed to file
func (s *Service) StoreRSS(chanID, rss string) error {
	return s.RSSFileStore.Save(chanID, rss)
}

// RemoveEntry deleted entry from store. Doesn't removes file
func (s *Service) RemoveEntry(entry ytfeed.Entry) error {
	if err := s.Store.ResetProcessed(entry); err != nil {
		return fmt.Errorf("failed to reset processed entry %s: %w", entry.VideoID, err)
	}
	if err := s.Store.Remove(entry); err != nil {
		return fmt.Errorf("failed to remove entry %s: %w", entry.VideoID, err)
	}
	return nil
}

// isNew checks if entry already processed
func (s *Service) isNew(entry ytfeed.Entry, fi FeedInfo) (ok bool, err error) {

	// check if entry already exists in store
	// this method won't work after migration to locally altered published ts but have to stay for now
	// to avoid false-positives on old entries what never got set with SetProcessed
	exists, exErr := s.Store.Exist(entry)
	if exErr != nil {
		return false, fmt.Errorf("failed to check if entry %s exists: %w", entry.VideoID, exErr)
	}
	if exists {
		return false, nil
	}

	// check if we already processed this entry.
	// this is needed to avoid infinite get/remove loop when the original feed is updated in place.
	// after migration to locally altered published ts, it is also the primary way to detect already processed entries
	found, _, procErr := s.Store.CheckProcessed(entry)
	if procErr != nil {
		log.Printf("[WARN] can't get processed status for %s, %+v", entry.VideoID, fi)
	}
	if procErr == nil && found {
		return false, nil
	}

	return true, nil
}

// isAllowed checks if entry matches all filters for the channel feed
func (s *Service) isAllowed(entry ytfeed.Entry, fi FeedInfo) (ok bool, err error) {

	matchedIncludeFilter := true
	if fi.Filter.Include != "" {
		matchedIncludeFilter, err = regexp.MatchString(fi.Filter.Include, entry.Title)
		if err != nil {
			return false, fmt.Errorf("failed to check if entry %s matches include filter: %w", entry.VideoID, err)
		}
	}

	matchedExcludeFilter := false
	if fi.Filter.Exclude != "" {
		matchedExcludeFilter, err = regexp.MatchString(fi.Filter.Exclude, entry.Title)
		if err != nil {
			return false, fmt.Errorf("failed to check if entry %s matches exclude filter: %w", entry.VideoID, err)
		}
	}

	return matchedIncludeFilter && !matchedExcludeFilter, nil
}

func (s *Service) isShort(file string) (bool, time.Duration) {
	if s.SkipShorts.Seconds() > 0 {
		// skip shorts if duration is less than SkipShorts
		duration := s.DurationService.File(file)
		if duration > 0 && duration < int(s.SkipShorts.Seconds()) {
			return true, time.Duration(duration) * time.Second
		}
	}
	return false, 0
}

// update sets entry file name and reset published ts
func (s *Service) update(entry ytfeed.Entry, file string, fi FeedInfo) ytfeed.Entry {
	entry.File = file

	// only reset time if published not too long ago
	// this is done to avoid initial set of entries added with a new channel to the top of the feed
	if time.Since(entry.Published) < time.Hour*24 {
		log.Printf("[DEBUG] reset published time for %s, from %s to %s (%v), %s",
			entry.VideoID, entry.Published.Format(time.RFC3339), time.Now().Format(time.RFC3339),
			time.Since(entry.Published), entry.String())
		entry.Published = time.Now() // reset published ts to prevent possible out-of-order entries
	} else {
		log.Printf("[DEBUG] keep published time for %s, %s", entry.VideoID, entry.Published.Format(time.RFC3339))
	}

	if !strings.Contains(entry.Title, fi.Name) { // if title doesn't contains channel name add it
		entry.Title = fi.Name + ": " + entry.Title
	}

	entry.Duration = s.DurationService.File(file)
	log.Printf("[DEBUG] updated entry: %s", entry.String())
	return entry
}

// removeOld deletes old entries from store and corresponding files
func (s *Service) removeOld(fi FeedInfo) int {
	removed := 0
	keep := s.keep(fi)
	files, err := s.Store.RemoveOld(fi.ID, keep+1)
	if err != nil { // even with error we get a list of files to remove
		log.Printf("[WARN] failed to remove some old meta data for %s, %v", fi.ID, err)
	}

	for _, f := range files {
		if e := os.Remove(f); e != nil {
			log.Printf("[WARN] failed to remove file %s: %v", f, e)
			continue
		}
		removed++
		log.Printf("[INFO] removed %s for %s (%s)", f, fi.ID, fi.Name)
	}
	return removed
}

func (s *Service) keep(fi FeedInfo) int {
	keep := s.KeepPerChannel
	if fi.Keep > 0 {
		keep = fi.Keep
	}
	return keep
}

func (s *Service) makeFileName(entry ytfeed.Entry) string {
	h := sha1.New()
	if _, err := h.Write([]byte(entry.UID())); err != nil {
		return uuid.New().String()
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// totalEntriesToKeep returns total number of entries to keep, summing all channels' keep values
func (s *Service) totalEntriesToKeep() (res int) {
	for _, fi := range s.Feeds {
		res += s.keep(fi)
	}
	return res
}

// countAllEntries returns total number of entries across all channels, respects keep settings
func (s *Service) countAllEntries() int {
	var result int
	for _, fi := range s.Feeds {
		if entries, err := s.Store.Load(fi.ID, s.keep(fi)); err == nil {
			result += len(entries)
		}
	}
	return result
}

// newestEntry returns the newest entry across all channels, respects keep settings
func (s *Service) newestEntry() ytfeed.Entry {
	entries := []ytfeed.Entry{}
	for _, fi := range s.Feeds {
		if recs, err := s.Store.Load(fi.ID, 1); err == nil {
			entries = append(entries, recs...)
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Published.After(entries[j].Published)
	})
	if len(entries) == 0 {
		return ytfeed.Entry{}
	}
	return entries[0]
}

// oldestEntry returns the oldest entry from all channels, respecting keep settings
func (s *Service) oldestEntry() ytfeed.Entry {
	entries := []ytfeed.Entry{}
	for _, fi := range s.Feeds {
		if recs, err := s.Store.Load(fi.ID, s.keep(fi)); err == nil {
			entries = append(entries, recs...)
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Published.Before(entries[j].Published)
	})
	if len(entries) == 0 {
		return ytfeed.Entry{}
	}
	return entries[0]
}

func (s *Service) updateMp3Tags(file string, entry ytfeed.Entry, fi FeedInfo) error {
	fh, err := id3v2.Open(file, id3v2.Options{Parse: false})
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", file, err)
	}
	defer fh.Close()

	fh.SetTitle(entry.Title)
	fh.SetArtist(entry.Author.Name)
	fh.SetAlbum(fi.Name)
	fh.SetGenre("podcast")
	fh.SetYear(entry.Published.Format("2006"))
	fh.AddTextFrame(fh.CommonID("Recording time"), fh.DefaultEncoding(), entry.Published.Format("20060102T150405"))

	if err = fh.Save(); err != nil {
		return fmt.Errorf("failed to close file %s: %w", file, err)
	}
	return nil
}

func (s *Service) execYtdlpUpdate(ctx context.Context, updCmd string) {
	log.Printf("[INFO] executing yt-dlp update command %s", s.YtDlpUpdCommand)
	cmd := exec.CommandContext(ctx, "sh", "-c", updCmd) // nolint
	cmd.Stdin = os.Stdin
	cmd.Stdout = log.ToWriter(log.Default(), "DEBUG")
	cmd.Stderr = log.ToWriter(log.Default(), "INFO")
	if err := cmd.Run(); err != nil {
		log.Printf("[WARN] failed to execute yt-dlp update command %s: %v", s.YtDlpUpdCommand, err)
	}
}

type stats struct {
	entries   int
	processed int
	added     int
	removed   int
	ignored   int
	skipped   int
}

func (st stats) String() string {
	return fmt.Sprintf("entries: %d, processed: %d, updated: %d, removed: %d, ignored: %d, skipped: %d",
		st.entries, st.processed, st.added, st.removed, st.ignored, st.skipped)
}
