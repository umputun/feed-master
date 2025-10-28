FROM umputun/baseimage:buildgo-latest as build

ARG GIT_BRANCH
ARG GITHUB_SHA
ARG CI

ENV CGO_ENABLED=0

ADD . /build
WORKDIR /build

RUN \
    if [ -z "$CI" ] ; then \
    echo "runs outside of CI" && version=$(git rev-parse --abbrev-ref HEAD)-$(git log -1 --format=%h)-$(date +%Y%m%dT%H:%M:%S); \
    else version=${GIT_BRANCH}-${GITHUB_SHA:0:7}-$(date +%Y%m%dT%H:%M:%S); fi && \
    echo "version=$version" && \
    cd app && go build -o /build/feed-master -ldflags "-X main.revision=${version} -s -w"



FROM umputun/baseimage:app-latest

# enables automatic changelog generation by tools like Dependabot
LABEL org.opencontainers.image.source="https://github.com/umputun/feed-master"

COPY --from=build /build/feed-master /srv/feed-master
COPY app/webapp /srv/webapp
RUN \
    chown -R app:app /srv && \
    chmod +x /srv/feed-master
RUN apk --no-cache add ca-certificates ffmpeg python3 py3-pip deno
RUN pip3 install --break-system-packages --no-cache-dir --no-deps -U yt-dlp bgutil-ytdlp-pot-provider
WORKDIR /srv

CMD ["/srv/feed-master"]
ENTRYPOINT ["/init.sh"]
