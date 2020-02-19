#!/usr/bin/env python3
# -*- coding: utf-8 -*-
from telethon.sync import TelegramClient, events
from optparse import OptionParser

parser = OptionParser()
parser.add_option("-s", "--session", type="string", help="Name of the session")
parser.add_option("-i", "--api_id", type="string", help="Telegram App API ID")
parser.add_option("-a", "--api_hash", type="string", help="Telegram App API Hash")

(options, args) = parser.parse_args()

with TelegramClient(options.session, options.api_id, options.api_hash) as client:
    client.disconnect()
