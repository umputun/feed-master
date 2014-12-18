build:
	docker build -t umputun/feed-master .

run:
	-docker rm -f feed-master
	docker run -d --name feed-master -v $(shell pwd)/config:/srv/config \
	-p 8099:8099 --link mongodb:mongodb umputun/feed-master

run-with-mongo:
	-docker run -d --name=mongodb -p 27017:27017 -v /data/mongo:/data/db mongo:latest mongod --smallfiles --noprealloc
	-docker rm -f feed-master
	docker run -d --name feed-master -v $(shell pwd)/config:/srv/config \
	-p 8099:8099 --link mongodb:mongodb umputun/feed-master


reload:
	docker restart feed-master

.PHONY: build run reload

