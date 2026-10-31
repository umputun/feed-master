---
worth: maybe
where: app/api/server.go:313
added: 2026-09-19
---
# Generated and served RSS can apply different item limits to the same channel

`regenerateRSSCtrl` builds `youtube.FeedInfo{ID: f.ID}` for every configured channel, dropping the
channel's `Keep`. `Service.keep` then falls back to `KeepPerChannel`, which is `youtube.max_per_channel`.
The `GET /yt/rss/{channel}` handler at `app/api/server.go:288` looks the channel up in the config and
passes the whole `FeedInfo`, so it uses the channel's effective `keep`. Automatic RSS writes after a
successful download (`app/youtube/service.go:339`) do the same.

So the two paths can produce different item counts for one channel, when their limits differ and enough
entries are stored. With `keep: 35`, `max_per_channel: 20` and at least 21 stored entries, `POST
/yt/rss/generate` writes 20 items and a later automatic rewrite writes up to 35.

The fix is to pass the configured `FeedInfo` in `regenerateRSSCtrl`, as the GET handler does. Whether
that is the right call depends on whether `max_per_channel` should keep any role at all, and nobody has
ruled on that, which is why this is `maybe`. The unconfigured-channel fallback in the GET handler uses
`max_per_channel` too.

Deferred during the documentation correction for #125, which exposed the split but did not resolve it.
