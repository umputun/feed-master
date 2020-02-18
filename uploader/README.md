# Feed Master Uploader

This Python-based Telegram client can upload files as a user, without 50MB limit.

1. [Register a new Telegram app](https://my.telegram.org/apps), get `api_id`, `api_hash`.

2. Set environment variables:

```bash
export API_ID="0123456"
export API_HASH="0123456789acbdefghijklmnopqrstuw"
export SESSION="uploader"
```

3. Create Telegram [session](https://telethon.readthedocs.io/en/latest/concepts/sessions.html) 
by running `auth.py` script, which will create `uploader.session` file:

```
cd uploader
python3 auth.py

Please enter your phone (or bot token): +12345678901
Please enter the code you received: 12345
Please enter your password: 
Invalid password. Please try again
Please enter your password: 
Signed in successfully as FirstName LastName
```

4. `feed-master` app expects additional environment variables:

```bash
export UPLOADER_ENABLED="true"
export PATH_TO_SCRIPT="/srv/uploader/uploader.py"
export API_ID="0123456"
export API_HASH="0123456789acbdefghijklmnopqrstuw"
export SESSION="uploader"
```