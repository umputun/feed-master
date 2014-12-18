# feed master
FROM debian:wheezy
MAINTAINER Umputun <feedmaster@umputun.com>


RUN \
 build_deps='binutils build-essential bzip2 cpp cpp-4.7 dpkg-dev fakeroot file g++ g++-4.7 gcc gcc-4.7' && \
 apt-get update && apt-get upgrade -y --no-install-recommends && \
 apt-get install -y python-pip && \
 apt-get autoremove -y && apt-get clean && \
 pip install feedparser plumbum pymongo && \
 apt-get purge -y --auto-remove $build_deps && \
 rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/* && \
 rm -rf /var/lib/{apt,dpkg,cache,log}

RUN \
 groupadd -r feedmaster && useradd -r -g feedmaster feedmaster && \
 mkdir /srv/data && \
 chown -R feedmaster:feedmaster /srv

VOLUME ["/srv/config"]

USER feedmaster
ADD src/feed-master.py /srv/feed-master.py
ADD exec.sh /srv/exec.sh
ADD src/config/__init__.py /srv/config/__init__.py

WORKDIR /srv
ENTRYPOINT ["/srv/exec.sh"]
