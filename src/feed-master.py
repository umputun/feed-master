#!/usr/bin/env python

__author__ = 'Umputun'

import logging
import sys
import time
from datetime import datetime
from email import utils
import os
import socket
import feedparser
from plumbum import cli
import pymongo

from config.config import feeds
from config.config import settings


root = logging.basicConfig(level=logging.INFO, format='%(asctime)s %(levelname)s - %(message)s', stream=sys.stdout)
NET_TIMEOUT = 15


class App(cli.Application):
    '''Feed-master utility'''

    PROGNAME = "feed-master"
    VERSION = "1.1"


@App.subcommand("update")
class UpdateItems(cli.Application):
    '''Update mongo from sources'''

    mongo_host = "127.0.0.1:27017"

    @cli.switch("--dbg", help="enable debug")
    def set_log_level(self):
        logging.root.setLevel(logging.DEBUG)

    @cli.switch(["--mongo", "-m"], str, help="set mongo server")
    def set_mongo(self, mongo_host):
        logging.debug("set mongo=%s", mongo_host)
        self.mongo_host = mongo_host

    def main(self):
        logging.info("feed loading initiated")
        socket.setdefaulttimeout(NET_TIMEOUT)

        if self.mongo_host.find(":") != -1:
            (mongo_ip, mongo_port) = self.mongo_host.split(":")
            mongo_client = pymongo.MongoClient(mongo_ip, int(mongo_port))
        else:
            mongo_client = pymongo.MongoClient(self.mongo_host)

        db = mongo_client["feed_master"]["feed"]
        db.create_index([("published", -1)])

        new_items = 0
        for (name, feed) in feeds:
            logging.debug("loading %s - %s", name, feed)
            d = feedparser.parse(feed)
            last_items = d['entries'][:settings['max_items_per_feed']]
            for item in last_items:
                enclosure = [x for x in item['links'] if x['rel'] == 'enclosure'][0]
                title = item['title']
                description = item['description']
                if db.find_one({"_id": enclosure['href']}):
                    logging.debug("already here, skip " + enclosure['href'])
                else:
                    pub_dtime = datetime.fromtimestamp(time.mktime(item['published_parsed']))
                    if pub_dtime > datetime.now() or abs((pub_dtime - datetime.now()).total_seconds()) < 2 * 60 * 60:
                        # for items in the future or close enough no now - reset timestamp
                        logging.debug("timestamp adjusted to now for %s - %s", title, pub_dtime)
                        pub_dtime = datetime.now()
                    mrec = {"_id": enclosure['href'], 'enclosure': enclosure, 'title': title,
                            'description': description, 'published': pub_dtime}
                    db.save(mrec)
                    logging.info("new item %s %s", title, pub_dtime)
                    new_items += 1
        logging.info("feed loading completed, new items=%d", new_items)


def format_datetime_rfc2822(dt):
    return utils.formatdate(time.mktime(dt.timetuple()))


@App.subcommand("generate")
class GenerateFeed(cli.Application):
    '''Generate RSS feed'''

    mongo_host = "127.0.0.1:27017"
    feed_file = "feed.xml"

    @cli.switch("--dbg", help="enable debug")
    def set_log_level(self):
        logging.root.setLevel(logging.DEBUG)

    @cli.switch(["--mongo", "-m"], str, help="set mongo server")
    def set_mongo(self, mongo_host):
        logging.debug("set mongo=%s", mongo_host)
        self.mongo_host = mongo_host

    @cli.switch(["--file", "-f"], str, help="set feed file")
    def set_feed_file(self, feed_file):
        logging.info("set feed file=%s", feed_file)
        self.feed_file = feed_file

    def main(self):
        logging.info("feed generation initiated")

        if self.mongo_host.find(":") != -1:
            (mongo_ip, mongo_port) = self.mongo_host.split(":")
            mongo_client = pymongo.MongoClient(mongo_ip, int(mongo_port))
        else:
            mongo_client = pymongo.MongoClient(self.mongo_host)
        db = mongo_client["feed_master"]["feed"]

        items = db.find().sort("published", -1).limit(settings['max_items_total'])
        last_date = format_datetime_rfc2822(db.find_one(sort=[("published", -1)])['published'])
        total_items = 0
        with open(self.feed_file + ".tmp", "w") as feed_file:

            rss_header = """
                <rss xmlns:geo="http://www.w3.org/2003/01/geo/wgs84_pos#" xmlns:content="http://purl.org/rss/1.0/modules/content/"
                xmlns:media="http://search.yahoo.com/mrss/" xmlns:yt="http://gdata.youtube.com/schemas/2007"
                xmlns:atom="http://www.w3.org/2005/Atom" xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd" version="2.0">
                \n<channel>\n
            """
            feed_file.write(rss_header)
            feed_file.write("<title>%s</title>\n" % settings['info']['title'].encode('utf-8'))
            feed_file.write("<description>%s</description>\n" % settings['info']['description'].encode('utf-8'))
            feed_file.write("<link>%s</link>\n" % settings['info']['link'].encode('utf-8'))
            feed_file.write("<pubDate>%s</pubDate>\n" % last_date)
            feed_file.write("<language>%s</language>\n" % settings['language'])
            feed_file.write("<generator>feed-master by Umputun</generator>\n")

            for item in items:
                try:
                    feed_file.write("<item>\n")
                    feed_file.write("<title>%s</title>\n" % item['title'].encode('utf-8'))
                    feed_file.write("<description>%s</description>\n" % item['description'].encode('utf-8'))
                    feed_file.write("<link>%s</link>\n" % item['_id'].encode('utf-8'))
                    feed_file.write("<pubDate>%s</pubDate>\n" % format_datetime_rfc2822(item['published']))
                    feed_file.write("<guid>%s</guid>\n" % item['_id'].encode('utf-8'))
                    feed_file.write('<enclosure length="%s" type="audio/mpeg" url="%s"/>' %
                                    (item['enclosure']['length'], item['enclosure']['href']))
                    feed_file.write("</item>\n")
                    total_items += 1
                except Exception, e:
                    logging.warn("failed to write %s, error=%s", item, e)

            feed_file.write('</channel>\n</rss>\n')

        os.rename(self.feed_file + '.tmp', self.feed_file)
        logging.info("feed generation completed, total feeds=%d, total items=%d", len(feeds), total_items)


if __name__ == "__main__":
    App.run()
