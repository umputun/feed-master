import os
import sys
from telethon import TelegramClient, sync
from telethon.tl.types import DocumentAttributeAudio
import eyed3


def send_uwp_podcast():
    try:

        assert len(sys.argv) > 4, 'Error command. ' \
                                  'Example: python uwp_send.py ' \
                                  '"123456" "3f3d6f7f63afc67d570def7e3da4165c" "@uwp_podcast" ' \
                                  '"/home/user/test.mp3" "first message" "- second message" "https://example.com"'

        api_id = sys.argv[1]
        api_hash = sys.argv[2]
        name_channel = sys.argv[3]
        filename = sys.argv[4]
        mess = '\n'.join(sys.argv[5:])

        audiofile = eyed3.load(filename)

        client = TelegramClient('session_name', api_id, api_hash)
        client.start()

        client.send_file(
            name_channel,
            file=filename,
            caption=mess,
            file_name=os.path.split(filename)[-1],
            use_cache=False,
            part_size_kb=512,
            attributes=[DocumentAttributeAudio(
                duration=int(audiofile.info.time_secs),
                title=audiofile.tag.title,
                performer=audiofile.tag.artist,
                voice=True)
            ]
        )

        client.disconnect()
    except Exception as ex:
        print(ex)


if __name__ == "__main__":
    send_uwp_podcast()
