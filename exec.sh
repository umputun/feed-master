#!/bin/bash

cd /srv/data
python -m SimpleHTTPServer 8099 &

while true
do
  /srv/feed-master.py update -m mongodb
  /srv/feed-master.py generate -m mongodb -f /srv/data/feed.xml
  sleep 600
done
