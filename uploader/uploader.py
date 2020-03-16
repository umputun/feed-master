#!/usr/bin/env python3
# -*- coding: utf-8 -*-
from telethon.sync import TelegramClient, events
from telethon.tl.types import DocumentAttributeAudio
import progressbar as pb
import eyed3
from sys import stdin, exit
from os import path
from optparse import OptionParser

parser = OptionParser()
parser.add_option("-s", "--session", type="string", help="Name of the session")
parser.add_option("-i", "--api_id", type="string", help="Telegram App API ID")
parser.add_option("-a", "--api_hash", type="string", help="Telegram App API Hash")
parser.add_option("-u", "--auth_only", action="store_true", help="Only create session file, don't upload file")
parser.add_option("-t", "--send_to", type="string", help="Channel, username or botname to send MP3 file to")
parser.add_option("-f", "--file_path", type="string", help="Path to MP3 file", metavar="FILE")
parser.add_option("-c", "--caption", type="string", help="Caption for Telegram audio file message")
parser.add_option("-m", "--parse_mode", type="string", default="html", help="Telegram message parse mode (html, md)")
parser.add_option("-p", "--show_progress_bar", action="store_true", default=False, help="Show progress bar")

(options, args) = parser.parse_args()

if options.auth_only:
    with TelegramClient(options.session, options.api_id, options.api_hash) as client:
        client.disconnect()
    print("Done")
    exit(0)

if options.show_progress_bar:
    widgets = ['Uploading: ', pb.Percentage(), ' ',
            pb.Bar(), ' ',
            pb.SimpleProgress(), ' ',
            pb.ETA()]
    bar = pb.ProgressBar(widgets=widgets, maxval=path.getsize(options.file_path))

def progress(sent, total):
    if options.show_progress_bar:
        bar.update(sent)

with TelegramClient(options.session, options.api_id, options.api_hash) as client:
    file = eyed3.load(options.file_path)
    title = "%s – %s" % (file.tag.title, file.tag.artist)

    if options.show_progress_bar:
        bar.start()

    client.parse_mode=options.parse_mode
    message = client.send_file(
        options.send_to,
        options.file_path,
        progress_callback=progress,
        attributes=[DocumentAttributeAudio(
            duration=int(file.info.time_secs),
            voice=None,
            title=file.tag.title,
            performer=file.tag.artist
        )],
        caption=options.caption
    )

    if options.show_progress_bar:
        bar.finish()

    client.disconnect()

    print(message)
