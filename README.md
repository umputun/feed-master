# Feed Master

Pulls multiple podcast feeds (RSS) and republishes as a common feed, properly sorted and podcast-client friendly.

## Run in docker (short version)

- Copy `docker-compose.yml` and adjust exposed port if needed
- Create `etc/fm.yml` (sample provided in `_example`)
- Start container with `docker-compose up -d`

## API

- `GET /rss/{name}` - returns feed-set for given name
- `GET /list` - returns list of feed-sets (json)

## Web UI

Web UI shows a list of items from generated RSS. It is available on `/feed/{name}`
