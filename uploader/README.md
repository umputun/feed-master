# Feed Master Uploader

This Python-based Telegram client can upload files as a user, without 50MB limit.

1. [Register a new Telegram app](https://my.telegram.org/apps), get `api_id`, `api_hash`.

2. `feed-master` app expects additional environment variables:

```bash
export UPLOADER_ENABLED="true"
export UPLOADER_PATH_TO_SCRIPT="/srv/uploader/uploader.py"
export UPLOADER_SESSION="uploader"
export UPLOADER_API_ID="0123456"
export UPLOADER_API_HASH="0123456789acbdefghijklmnopqrstuw"
```

3. Create Telegram [session](https://telethon.readthedocs.io/en/latest/concepts/sessions.html) 
by running `uploader.py` script with `--auth_only` flag, which will create `uploader.session` file:

```bash
# locally

python3 uploader/uploader.py --session $UPLOADER_SESSION --api_id $UPLOADER_API_ID --api_hash $UPLOADER_API_HASH --auth_only

# or using Docker exec

docker exec -it feed-master sh -c 'python3 uploader/uploader.py --session $UPLOADER_SESSION --api_id $UPLOADER_API_ID --api_hash $UPLOADER_API_HASH --auth_only'

# or using Docker Compose exec

docker-compose exec feed-master sh -c 'python3 uploader/uploader.py --session $UPLOADER_SESSION --api_id $UPLOADER_API_ID --api_hash $UPLOADER_API_HASH --auth_only'
```
