// Package youtube provides loading audio from video files for given youtube channels
package youtube

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/umputun/feed-master/app/youtube/channel"
)

//go:generate moq -out mocks/downloader.go -pkg mocks -skip-ensure -fmt goimports . DownloaderService
//go:generate moq -out mocks/channel.go -pkg mocks -skip-ensure -fmt goimports . ChannelService
//go:generate moq -out mocks/store.go -pkg mocks -skip-ensure -fmt goimports . StoreService

// Service loads audio from youtube channels
type Service struct {
	Channels       []ChannelInfo
	Downloader     DownloaderService
	ChannelService ChannelService
	Store          StoreService
	CheckDuration  time.Duration
	KeepPerChannel int
}

// ChannelInfo is a pait of channel ID and name
type ChannelInfo struct {
	Name string
	ID   string
}

// DownloaderService is an interface for downloading audio from youtube
type DownloaderService interface {
	Get(ctx context.Context, id string, fname string) (file string, err error)
}

// ChannelService is an interface for getting channel entries, i.e. the list of videos
type ChannelService interface {
	Get(ctx context.Context, chanID string) ([]channel.Entry, error)
}

// StoreService is an interface for storing and loading metadata about downloaded audio
type StoreService interface {
	Save(entry channel.Entry) (bool, error)
	Load(channelID string, max int) ([]channel.Entry, error)
	Exist(entry channel.Entry) (bool, error)
	RemoveOld(channelID string, keep int) ([]string, error)
}

// Do is a blocking function that downloads audio from youtube channels and updates metadata
func (s *Service) Do(ctx context.Context) error {

	tick := time.NewTicker(s.CheckDuration)
	defer tick.Stop()

	if err := s.procChannels(ctx); err != nil {
		return errors.Wrap(err, "failed to process channels")
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
			if err := s.procChannels(ctx); err != nil {
				return errors.Wrap(err, "failed to process channels")
			}
		}
	}
}

func (s *Service) procChannels(ctx context.Context) error {
	for _, chanInfo := range s.Channels {
		entries, err := s.ChannelService.Get(ctx, chanInfo.ID)
		if err != nil {
			log.Printf("[WARN] failed to get channel entries for %s: %s", chanInfo.ID, err)
			continue
		}

		for _, entry := range entries {
			exists, err := s.Store.Exist(entry)
			if err != nil {
				return errors.Wrapf(err, "failed to check if entry %s exists", entry.VideoID)
			}
			if exists {
				continue
			}
			log.Printf("[INFO] new entry %s, %s, %s", entry.VideoID, entry.Title, chanInfo.Name)
			file, err := s.Downloader.Get(ctx, entry.VideoID, uuid.New().String())
			if err != nil {
				log.Printf("[WARN] failed to download %s: %s", entry.VideoID, err)
				continue
			}
			log.Printf("[DEBUG] downloaded %s (%s) to %s, channel: %+v", entry.VideoID, entry.Title, file, chanInfo)
			entry.File = file
			ok, err := s.Store.Save(entry)
			if err != nil {
				return errors.Wrapf(err, "failed to save entry %+v", entry)
			}
			if !ok {
				log.Printf("[WARN] attempt to save dup entry %+v", entry)
			}
			log.Printf("[INFO] saved %s (%s) to %s, channel: %+v", entry.VideoID, entry.Title, file, chanInfo)
		}

		// removed old entries and files
		files, err := s.Store.RemoveOld(chanInfo.ID, s.KeepPerChannel)
		if err != nil {
			return errors.Wrapf(err, "failed to remove old meta data for %s", chanInfo.ID)
		}
		for _, f := range files {
			if err := os.Remove(f); err != nil {
				log.Printf("[WARN] failed to remove file %s: %s", f, err)
				continue
			}
			log.Printf("[INFO] removed %s for %s (%s)", f, chanInfo.ID, chanInfo.Name)
		}
	}
	return nil
}
