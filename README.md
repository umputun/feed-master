# Feed Master [![Build Status](https://github.com/umputun/feed-master/workflows/build/badge.svg)](https://github.com/umputun/feed-master/actions) [![Coverage Status](https://coveralls.io/repos/github/umputun/feed-master/badge.svg?branch=master)](https://coveralls.io/github/umputun/feed-master?branch=master) [![Docker Automated build](https://img.shields.io/docker/automated/umputun/feed-master)](https://hub.docker.com/r/umputun/feed-master)

Feed-Master is a service that aggregates and publishes RSS feeds. It can pull multiple feeds from different sources and publish them to a single feed. The service normalizing all the feeds to make sure the combined feed is valid, compatible with podcast clients and compatible with RSS 2.0 specification. 

In addition to making RSS feeds, Feed-Master can also publish updates to both twitter and telegram. In case of telegram the actual mp3 audio file is published too. In case of twitter the mp3 audio file is published as a tweet with a link to the original audio file and with the episode info, like title and/or description.


Feed-Master supports extracting audio from youtube channels and use it to make the final feed. The service uses [yt-dlp](https://github.com/yt-dlp/yt-dlp) to pull videos and [ffmpeg](https://www.ffmpeg.org/) for audio extraction. In this mode feed-master serves the audio files in addition to the generated RSS feed.

## Run in docker (short version)

- Copy `docker-compose.yml` and adjust exposed port if needed
- Create `etc/fm.yml` (samples provided in `_example`)
- Start container with `docker-compose up -d feed-master`

### Application parameters

| Command line     | Environment         | Default                    | Description                               |
|------------------|---------------------|----------------------------|-------------------------------------------|
| db               | FM_DB               | `var/feed-master.bdb`      | bolt db file                              |
| conf             | FM_CONF             | `feed-master.yml`          | config file (yml)                         |
| feed             | FM_FEED             |                            | single feed, overrides config             |
| update-interval  | UPDATE_INTERVAL     | `1m`                       | update interval, overrides config         |
| telegram_chan    | TELEGRAM_CHAN       |                            | single telegram channel, overrides config |
| telegram_server  | TELEGRAM_SERVER     | `https://api.telegram.org` | telegram bot api server                   |
| telegram_token   | TELEGRAM_TOKEN      |                            | telegram token                            |
| telegram_timeout | TELEGRAM_TIMEOUT    | `1m`                       | telegram timeout                          |
| consumer-key     | TWI_CONSUMER_KEY    |                            | twitter consumer key                      |
| consumer-secret  | TWI_CONSUMER_SECRET |                            | twitter consumer secret                   |
| access-token     | TWI_ACCESS_TOKEN    |                            | twitter access token                      |
| access-secret    | TWI_ACCESS_SECRET   |                            | twitter access secret                     |
| template         | TEMPLATE            | `{{.Title}} - {{.Link}}`   | twitter message template                  |
| dbg              | DEBUG               | `false`                    | debug mode                                |

## API

- `GET /rss/{name}` - returns feed-set for given feed name
- `GET /list` - returns list of feed-sets (json)
- `GET /image/{name}` - returns image for given feed name
- `GET /feed/{name}/sources` - returns list of sources for given feed name
- `GET /yr/rss/{channel}` - return RSS feed for given youtube channel

## Web UI

Web UI shows a list of items from generated RSS. It is available on `/feeds` or, for the particular output feed on `/feed/{name}`

## Telegram notifications

By default, (with only `TELEGRAM_TOKEN` provided) Telegram notifications will be sent using standard Bot API which has a limit of [50Mb](https://core.telegram.org/bots/api#sending-files) for audio file upload.

You can provide `TELEGRAM_API_ID` and `TELEGRAM_API_HASH` (from [here](https://my.telegram.org/apps)) to `telegram-bot-api` service in docker-compose.yml and uncomment `TELEGRAM_SERVER` for `feed-master`, then it would use the local bot api server to raise audio file upload limit from 50Mb [to 2000Mb](https://core.telegram.org/bots/api#using-a-local-bot-api-server).

To use local telegram bot api server, use `docker-compose up -d` command instead of `docker-compose up -d feed-master`.

