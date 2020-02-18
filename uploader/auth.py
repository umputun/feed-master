#!/usr/bin/env python3
# -*- coding: utf-8 -*-
from telethon.sync import TelegramClient, events
from os import environ

api_id = environ.get('API_ID')
api_hash = environ.get('API_HASH')
session = environ.get('SESSION')

with TelegramClient(session, api_id, api_hash) as client:
    client.disconnect()
