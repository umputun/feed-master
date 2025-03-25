# Feed Master [![Build Status](https://github.com/umputun/feed-master/workflows/build/badge.svg)](https://github.com/umputun/feed-master/actions) [![Coverage Status](https://coveralls.io/repos/github/umputun/feed-master/badge.svg?branch=master)](https://coveralls.io/github/umputun/feed-master?branch=master) [![Docker Automated build](https://img.shields.io/docker/automated/umputun/feed-master)](https://hub.docker.com/r/umputun/feed-master)


Feed-Master is a service that aggregates and publishes RSS feeds from multiple sources into a single feed. It normalizes the feeds to ensure that they are valid, compatible with podcast clients, and compliant with the RSS 2.0 specification. This allows users to access all of their desired content in a single, easy-to-use feed.

In addition to aggregating RSS feeds, Feed-Master can also publish updates to social media platforms such as Twitter and Telegram. For Telegram, the actual audio file is published, while for Twitter, a link to the original audio file is included in the tweet along with episode information like the title and description.

Feed-Master also supports extracting audio from YouTube channels and using it to create the final feed. The service uses tools like  [yt-dlp](https://github.com/yt-dlp/yt-dlp) and [ffmpeg](https://www.ffmpeg.org/) to pull videos and extract the audio, respectively. In this mode, Feed-Master serves the audio files in addition to the generated RSS feed, providing users with even more options for accessing and consuming content.

## Run in docker (short version)

- Copy `docker-compose.yml` and adjust exposed port if needed
- Create `etc/fm.yml` (samples provided in `_example`)
- Start container with `docker-compose up -d feed-master`

_example of docker-compose.yml available in [_example](https://github.com/umputun/feed-master/tree/master/_example)_

## Main application parameters

| Command line | Environment  | Default               | Description                           |
|--------------|--------------|-----------------------|---------------------------------------|
| db           | FM_DB        | `var/feed-master.bdb` | bolt db file                          |
| conf         | FM_CONF      | `feed-master.yml`     | config file (yml)                     |
| admin-passwd | ADMIN_PASSWD | `none` (disabled)     | admin password for protected endpoint |
| dbg          | DEBUG        | `false`               | debug mode                            |


## Configuration

Usually, feed-master configuration is stored in `feed-master.yml` file. It is a yaml file with the following structure:

```yaml
feeds:
  yt-example: # feed name, can be repeated for multiple source feeds
    title: Some cool channels # feed title
    description: an example of youtube-based podcas # feed description
    link: http://localhost:8080/feed/yt-example # link to the source site
    language: "ru-ru" # feed language
    author: "Someone" # feed author, default "Feed Master"
    owner_email: "blah@example.com" # feed owner email, used in various services (i.e. spotify) to confirm RSS submission
    image: images/yt-example.png # feed image, used in generated RSS as podcast thumbnail
    filter: 
      - Title: "something" # filter from the feed, can be regexp or string
      - Invert: true # invert filter (acts as "only"), default false
    sources: # list of sources, each source is a name of and the source RSS feed
      - {name: "Точка", url: http://localhost:8080/yt/rss/PLZVQqcKxEn_6YaOniJmxATjODSVUbbMkd}
      - {name: "Живой Гвоздь", url: http://localhost:8080/yt/rss/UCWAIvx2yYLK_xTYD4F2mUNw}
      - {name: "Дилетант", url: http://localhost:8080/yt/rss/UCuIE7-5QzeAR6EdZXwDRwuQ}


youtube: # youtube configuration, optional
  base_url: http://localhost:8080/yt/media # base url for youtube media
  dl_template: yt-dlp --extract-audio --audio-format=mp3 --audio-quality=0 -f m4a/bestaudio "https://www.youtube.com/watch?v={{.ID}}" --no-progress -o {{.FileName}} # template for youtube-dl
  base_chan_url: "https://www.youtube.com/feeds/videos.xml?channel_id=" # base url for youtube channel
  base_playlist_url: "https://www.youtube.com/feeds/videos.xml?playlist_id=" # base url for youtube playlist
  update: 60s # update interval for youtube feeds
  skip_shorts: 120s # skip videos (and audios) shorter than this value, optional
  max_per_channel: 2 # max number of the latest videos per yt channel to download and process
  files_location: ./var/yt # location for downloaded youtube files
  rss_location: ./var/rss # location for generated youtube channel's RSS
  channels: # list of youtube channels to download and process
      # id: channel or playlist id, name: channel or playlist name, type: "channel" or "playlist", 
      # lang: language of the channel, keep: override default keep value
      # filter: criteria to include and exclude videos, can be regex
      - {id: UCWAIvx2yYLK_xTYD4F2mUNw, name: "Живой Гвоздь", lang: "ru-ru"}
      - {id: UCuIE7-5QzeAR6EdZXwDRwuQ, name: "Дилетант", type: "channel", lang: "ru-ru", "keep": 10}
      - {id: PLZVQqcKxEn_6YaOniJmxATjODSVUbbMkd, name: "Точка", type: "playlist", lang: "ru-ru", filter: {include: "ТОЧКА", exclude: "STAR'цы Live"}} 
  ytdlp_update: 
    interval: 24h # update interval for yt-dlp. If not set, yt-dlp will not be updated 
    command: "pip3 install --break-system-packages -U yt-dlp" # update yt-dlp command

system: # system configuration
  update: 1m # update interval for checking source feeds
  http_response_timeout: 30s # http response timeout
  max_per_feed: 10 # max items per feed to be processed and inclueded in the final RSS
  max_total: 50 # max total items to be included in the final RSS
  max_keep: 1000 # max items to be kept in the internal database 
  base_url: http://localhost:8080 # base url for the generated RSS and media files
```

_see [examples](https://github.com/umputun/feed-master/tree/master/_example/etc) for more details._

### Single-feed configuration

For a very simple configuration, command-line only configuration is available. In this case only a single source feed is allowed and yt processing is disabled.  The command-line configuration is the following:

| Command line     | Environment         | Default                    | Description                               |
|------------------|---------------------|----------------------------|-------------------------------------------|
| feed             | FM_FEED             |                            | single feed, overrides config             |
| update-interval  | UPDATE_INTERVAL     | `1m`                       | update interval, overrides config         |
| telegram_chan    | TELEGRAM_CHAN       |                            | single telegram channel, overrides config |

Name of the channel or numeric ChatID could be used, [here](https://remark42.com/docs/configuration/telegram/#notifications-for-administrators) are the instructions on obtaining the ChatID. To be able to post messages to the channel, bot must be added as administrator with Post permission.

All this command-line mode is good for - process a single feed, send a telegram message and send a tweet on each new item.

### Notifications

In both configuration modes, user can specify a list of telegram and twitter accounts to be notified.

| Command line     | Environment         | Default                    | Description                               |
|------------------|---------------------|----------------------------|-------------------------------------------|
| telegram_server  | TELEGRAM_SERVER     | `https://api.telegram.org` | telegram bot api server                   |
| telegram_token   | TELEGRAM_TOKEN      |                            | telegram token                            |
| telegram_timeout | TELEGRAM_TIMEOUT    | `1m`                       | telegram timeout                          |
| consumer-key     | TWI_CONSUMER_KEY    |                            | twitter consumer key                      |
| consumer-secret  | TWI_CONSUMER_SECRET |                            | twitter consumer secret                   |
| access-token     | TWI_ACCESS_TOKEN    |                            | twitter access token                      |
| access-secret    | TWI_ACCESS_SECRET   |                            | twitter access secret                     |
| template         | TEMPLATE            | `{{.Title}} - {{.Link}}`   | twitter message template                  |


## API

_See [requests.http](https://github.com/umputun/feed-master/blob/master/requests.http)_

### public endpoints

- `GET /rss/{name}` - returns feed-set for given feed name
- `GET /list` - returns list of feed-sets (json)
- `GET /image/{name}` - returns image for given feed name
- `GET /feed/{name}/sources` - returns list of sources for given feed name
- `GET /yt/rss/{channel}` - return RSS feed for given youtube channel

### admin endpoints

- `POST /yt/rss/generate` - regenerate RSS feed for all youtube channels
- `DELETE /yt/entry/{channel}/{video}` - delete youtube entry from internal database and remove it from RSS feed

## Web UI

Web UI shows a list of items from generated RSS. It is available on `/feeds` or, for the particular output feed on `/feed/{name}`

## Telegram notifications details

By default, (with only `TELEGRAM_TOKEN` provided) Telegram notifications will be sent using standard Bot API which has a limit of [50Mb](https://core.telegram.org/bots/api#sending-files) for audio file upload.

You can provide `TELEGRAM_API_ID` and `TELEGRAM_API_HASH` (from [here](https://my.telegram.org/apps)) to `telegram-bot-api` service in docker-compose.yml and uncomment `TELEGRAM_SERVER` for `feed-master`, then it would use the local bot api server to raise audio file upload limit from 50Mb [to 2000Mb](https://core.telegram.org/bots/api#using-a-local-bot-api-server).

To use local telegram bot api server, use `docker-compose up -d` command instead of `docker-compose up -d feed-master`.
