# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build, Test, Lint Commands
```bash
# Run tests
go test -race -v ./...                            # Run all tests from root
go test -race -v ./app/...                        # Run app tests
go test -race -v ./app/proc                       # Test specific package
go test -race -v ./app/proc -run TestStore       # Run specific test

# Lint code
golangci-lint run ./...                           # Lint entire codebase from root
golangci-lint run ./app/...                       # Lint app directory

# Build application
cd app && go build -o feed-master                 # Build binary
docker build -t feed-master .                      # Build Docker image

# Format and normalize
gofmt -s -w $(find . -type f -name "*.go" -not -path "./vendor/*")
goimports -w $(find . -type f -name "*.go" -not -path "./vendor/*")
```

## High-Level Architecture

Feed Master is a Go service that aggregates RSS feeds and YouTube content into unified feeds:

- **app/main.go**: Entry point with CLI flags, initializes Processor and Server
- **app/proc**: Core feed processing logic
  - `Processor`: Orchestrates feed fetching, filtering, and notifications
  - `Store`: BoltDB persistence layer for feed items
  - `Telegram`/`Twitter`: Notification handlers
- **app/feed**: RSS feed parsing and generation utilities
- **app/youtube**: YouTube channel/playlist processing
  - `Service`: Downloads videos as audio, manages channel RSS generation
  - `feed.Downloader`: Handles yt-dlp interactions
  - `store.BoltDB`: Persists YouTube metadata
- **app/api**: HTTP endpoints for RSS feeds and admin operations
  - Public: `/rss/{name}`, `/list`, `/yt/rss/{channel}`
  - Admin: `/yt/rss/generate`, `/yt/entry/{channel}/{video}` (DELETE)
- **app/config**: YAML configuration loading and validation

## Key Design Patterns

- **Feed Aggregation**: Multiple source feeds → normalized → single output feed
- **YouTube Integration**: Uses yt-dlp for audio extraction, serves files via HTTP
- **Storage**: BoltDB for both feed items and YouTube metadata
- **Notifications**: Template-based messages to Telegram/Twitter on new items
- **Concurrent Processing**: Uses go-pkgz/syncs for controlled parallelism
- **Error Handling**: pkg/errors for wrapping, lgr for structured logging

## Testing Patterns

Tests use testify with table-driven patterns:
```go
func TestFeature(t *testing.T) {
    tests := []struct {
        name    string
        input   Type
        want    Result
        wantErr bool
    }{
        {"case 1", input1, expected1, false},
        {"error case", badInput, nil, true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```

## Configuration Structure

Config loaded from YAML (see _example/etc/):
- `feeds`: Named feed configurations with sources, filters, notifications
- `youtube`: Channel definitions, download settings, file locations
- `system`: Update intervals, limits, base URL

## Dependencies

- **Web**: chi/v5 router with go-pkgz/rest middlewares
- **Storage**: etcd.io/bbolt
- **Testing**: stretchr/testify
- **YouTube**: External yt-dlp binary
- **Notifications**: tucnak/telebot.v2, ChimeraCoder/anaconda

## Important Testing Note

The processor tests in `app/proc` may fail if test data contains dates older than 1 year. The processor skips RSS items older than 1 year (see `processor.go:83`). If tests fail with "no bucket for feed1" errors:

1. Check the dates in `app/proc/testdata/rss1.xml` and `app/proc/testdata/rss2.xml`
2. Update the year in `<pubDate>` tags to be within the last year
3. Example: Change `<pubDate>Sat, 19 Mar 2024 19:35:46 EST</pubDate>` to `<pubDate>Sat, 19 Mar 2025 19:35:46 EST</pubDate>`

## Production Environment

- **Server**: master.feed-master.com (direct SSH access)
- **Location**: `/srv`
- **Deployment**: Docker Compose
- **Config on server**: `/srv/etc/fm.yml`
- **Config source**: `/dev.umputun/system/hosts/master.feed-master.com/etc/fm.yml`
- **Deployment Makefile**: `/dev.umputun/system/hosts/master.feed-master.com/Makefile`

## Deployment Commands

Key Makefile targets in `/dev.umputun/system/hosts/master.feed-master.com/Makefile`:
- `make restart-feed-master` - Deploy config and restart (syncs fm.yml + cookies, pulls images, restarts feed-master and nginx)
- `make deploy` - Full deployment (syncs etc/, var/, docker-compose.yml, pulls and starts)
- `make update` - Pull images and restart without syncing config

## Troubleshooting

**Check logs on production**:
```bash
ssh master.feed-master.com "docker logs feed-master 2>&1 | tail -100"
```

**YouTube download issues** (common problem):
- Issues are often related to yt-dlp: https://github.com/yt-dlp/yt-dlp
- yt-dlp template configured in `fm.yml` under `youtube.dl_template`

**yt-dlp configuration flags** (in `dl_template`):
- `--remote-components ejs:github` - Required for JS challenge solving (uses deno runtime in container)
- `--extractor-args "youtubepot-bgutilhttp:base_url=http://bgutil-provider:4416"` - bgutil PO token provider for bot detection bypass
- Format selector `bestaudio[ext=m4a]/bestaudio/best` preferred over `bestaudio*/best` to avoid HLS 403 errors
- `--cookies /srv/etc/yt-cookies.txt` - YouTube authentication cookies

**Cookie file format**:
- Must use Netscape cookie format with **TAB separators** (not spaces)
- Common cause of yt-dlp auth failures: cookies saved with spaces instead of tabs
- Cookie file location: `/srv/etc/yt-cookies.txt` on server, source at `hosts/master.feed-master.com/etc/yt-cookies.txt`