# Feed Master Uploader

This Python-based Telegram client can upload files as a user, without 50MB limit.

1. [Register a new Telegram app](https://my.telegram.org/apps), get `api_id`, `api_hash`.

2. Create Telegram [session](https://telethon.readthedocs.io/en/latest/concepts/sessions.html) 
by running `auth.py` script, which will create `uploader.session` file:

```
cd uploader
python3 auth.py --session uploader --api_id 0123456 --api_hash 0123456789acbdefghijklmnopqrstuw

# or: python3 auth.py -s uploader -i 0123456 -a 0123456789acbdefghijklmnopqrstuw

Please enter your phone (or bot token): +12345678901
Please enter the code you received: 12345
Please enter your password: 
Invalid password. Please try again
Please enter your password: 
Signed in successfully as FirstName LastName
```

3. `feed-master` app expects additional environment variables:

```bash
export UPLOADER_ENABLED="true"
export UPLOADER_PATH_TO_SCRIPT="/srv/uploader/uploader.py"
export UPLOADER_API_ID="0123456"
export UPLOADER_API_HASH="0123456789acbdefghijklmnopqrstuw"
export UPLOADER_SESSION="uploader"
```
