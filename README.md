# Feed Master [![Build Status](https://github.com/umputun/feed-master/workflows/build/badge.svg)](https://github.com/umputun/feed-master/actions) [![Go Report Card](https://goreportcard.com/badge/github.com/umputun/feed-master)](https://goreportcard.com/report/github.com/umputun/feed-master) [![Coverage Status](https://coveralls.io/repos/github/umputun/feed-master/badge.svg?branch=master)](https://coveralls.io/github/umputun/feed-master?branch=master)

Pulls multiple podcast feeds (RSS) and republishes as a common feed, properly sorted and podcast-client friendly. Optionally posts the new items to telegram's channel.

## Run in docker (short version)

- Copy `docker-compose.yml` and adjust exposed port if needed
- Create `etc/fm.yml` (sample provided in `_example`)
- Start container with `docker-compose up -d`

## API

- `GET /rss/{name}` - returns feed-set for given name
- `GET /list` - returns list of feed-sets (json)

## Web UI

Web UI shows a list of items from generated RSS. It is available on `/feed/{name}`
