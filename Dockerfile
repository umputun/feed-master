FROM umputun/baseimage:buildgo-latest as build

ARG COVERALLS_TOKEN
ARG CI
ARG GIT_BRANCH
ARG SKIP_TEST

ENV GOFLAGS="-mod=vendor"

ADD . /build/feed-master
WORKDIR /build/feed-master

# run tests and linters
RUN \
    if [ -z "$SKIP_TEST" ] ; then \
    go test -timeout=30s  ./... && \
    golangci-lint run ; \
    else echo "skip tests and linter" ; fi

RUN \
    if [ -z "$CI" ] ; then \
    echo "runs outside of CI" && version=$(/script/git-rev.sh); \
    else version=${GIT_BRANCH}-${GITHUB_SHA:0:7}-$(date +%Y%m%dT%H:%M:%S); fi && \
    echo "version=$version" && \
    go build -o feed-master -ldflags "-X main.revision=${version} -s -w" ./app


FROM umputun/baseimage:app-latest

COPY --from=build /build/feed-master/feed-master /srv/feed-master
COPY app/webapp /srv/webapp
RUN \
    chown -R app:app /srv && \
    chmod +x /srv/feed-master

WORKDIR /srv

CMD ["/srv/feed-master"]
ENTRYPOINT ["/init.sh"]
