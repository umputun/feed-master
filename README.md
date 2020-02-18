# Feed Master [![Build Status](https://github.com/umputun/feed-master/workflows/build/badge.svg)](https://github.com/umputun/feed-master/actions) [![Go Report Card](https://goreportcard.com/badge/github.com/umputun/feed-master)](https://goreportcard.com/report/github.com/umputun/feed-master) [![Coverage Status](https://coveralls.io/repos/github/umputun/feed-master/badge.svg?branch=master)](https://coveralls.io/github/umputun/feed-master?branch=master)

Pulls multiple podcast feeds (RSS) and republishes as a common feed, properly sorted and podcast-client friendly. Optionally posts the new items to telegram's channel.

## Run in docker (short version)

- Copy `docker-compose.yml` and adjust exposed port if needed
- Create `etc/fm.yml` (sample provided in `_example`)
- Start container with `docker-compose up -d`

### Application parameters

| Command line     | Environment       | Default               | Description                         |
| -----------------| ------------------| ----------------------| ----------------------------------- |
| db               | FM_DB             | `var/feed-master.bdb` | bolt db file                        |
| conf             | FM_CONF           | `feed-master.yml`     | config file (yml)                   |
| feed             | FM_FEED           |                 | single feed, overrides config             |
| update-interval  | UPDATE_INTERVAL   | `1m`            | update interval, overrides config         |
| telegram_chan    | TELEGRAM_CHAN     |                 | single telegram channel, overrides config |
| telegram_token   | TELEGRAM_TOKEN    |                 | telegram token           |
| telegram_timeout | TELEGRAM_TIMEOUT  | `1m`            | telegram timeout         |
| consumer-key     | TWI_CONSUMER_KEY  |                 | twitter consumer key     |
| consumer-secret  | TWI_CONSUMER_SECRET |               | twitter consumer secret  |
| access-token     | TWI_ACCESS_TOKEN  |                 | twitter access token     |
| access-secret    | TWI_ACCESS_SECRET |                 | twitter access secret    |
| template         | TEMPLATE | `{{.Title}} - {{.Link}}` | twitter message template |
| dbg              | DEBUG             | `false`         | debug mode               |

## API

- `GET /rss/{name}` - returns feed-set for given name
- `GET /list` - returns list of feed-sets (json)

## Web UI

Web UI shows a list of items from generated RSS. It is available on `/feed/{name}`
