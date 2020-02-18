#!/usr/bin/env python3
# -*- coding: utf-8 -*-
from telethon.sync import TelegramClient, events
from telethon.tl.types import DocumentAttributeAudio
import progressbar as pb
import eyed3
from sys import stdin
from os import environ, path

api_id = environ.get('API_ID')
api_hash = environ.get('API_HASH')
session = environ.get('SESSION')
file_path = environ.get('FILE_PATH')
send_to = environ.get('SEND_TO')
caption = environ.get('CAPTION')
parse_mode = environ.get('PARSE_MODE')
show_progress_bar = environ.get('SHOW_PROGRESS_BAR') in ["true", "1", "y", "yes"]

if show_progress_bar:
    widgets = ['Uploading: ', pb.Percentage(), ' ',
            pb.Bar(), ' ',
            pb.SimpleProgress(), ' ',
            pb.ETA()]
    bar = pb.ProgressBar(widgets=widgets, maxval=path.getsize(file_path))

def progress(sent, total):
    if show_progress_bar:
        bar.update(sent)

with TelegramClient(session, api_id, api_hash) as client:
    file = eyed3.load(file_path)
    title = "%s – %s" % (file.tag.title, file.tag.artist)

    if show_progress_bar:
        bar.start()

    client.parse_mode=parse_mode
    message = client.send_file(
        send_to,
        file_path,
        progress_callback=progress,
        attributes=[DocumentAttributeAudio(
            duration=int(file.info.time_secs),
            voice=None,
            title=file.tag.title,
            performer=file.tag.artist
        )],
        caption=caption
    )

    if show_progress_bar:
        bar.finish()

    client.disconnect()

    print(message)
