# Feed Master

Pulls multiple podcast feeds (RSS) and republishes as a common feed.

## Run directly
- Edit config/config.py (sample provided)
- update feeds `feed-master.py update -m <mongodb-host>`
- publish merged rss file `feed-master.py generate -m <mongo-host> -f /path/to/feed.xml

## Run in docker (short version)
- make build
- make run
- hit http://ip:8099/feed.xml

## Run in docker (long version)
- build `docker build -t umputun/feed-master .`
- run dockerized mongo `docker run -d --name=mongodb -p 27017:27017 mongo`
- run feed master linked to mongo `docker run -d --name feed-master -v /path/to/config:/srv/confir -p 8099:8099 --link mongodb:mongodb umputun/feed-master`

