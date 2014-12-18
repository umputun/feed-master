# Feed Master

Pulls multiple podcast feeds (RSS) and republishes as a common feed, properly sorted and podcast-client friendly.

## Run directly
- Edit config/config.py (sample provided)
- update feeds `feed-master.py update -m <mongo-host>`
- publish merged rss file `feed-master.py generate -m <mongo-host> -f /path/to/feed.xml

## Run in docker (short version)
- make build
- make run-with-mongo
- hit http://localhost:8099/feed.xml

## Run in docker (longer version)
- build `docker build -t umputun/feed-master .`
- run dockerized mongo `docker run -d --name=mongodb -p 27017:27017 mongo`
- run feed master linked to mongo `docker run -d --name feed-master -v /path/to/config:/srv/config -p 8099:8099 --link mongodb:mongodb umputun/feed-master`

## Notes
- feed.xml served via exec.sh with SimpleHTTPServer. Good enough for single-user or feed it to feedburner. If you need some real server - expose /srv/data to your host and serve it with nginx/apache.
- feed master runs in 10mins loops, see `sleep 600` in exec.sh
- making mongo persistent may be a good idea. Makefile has run-with-mongo target doing this.
- change of config.py requires reload, i.e. `make reload` or `docker restart feed-master` for dockerized version.
- you can get feed-master from docker hub `docker pull umputun/feed-master`
