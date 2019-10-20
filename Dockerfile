FROM umputun/baseimage:buildgo-latest as build

ARG COVERALLS_TOKEN
ARG CI
ARG GIT_BRANCH

ENV GOFLAGS="-mod=vendor"

ADD . /build/feed-master
WORKDIR /build/feed-master

# run tests
RUN cd app && go test ./...

# linters
RUN golangci-lint run --out-format=tab --disable-all --tests=false --enable=interfacer --enable=unconvert --enable=megacheck \
    --enable=structcheck --enable=gocyclo --enable=dupl --enable=misspell --enable=maligned --enable=unparam \
    --enable=varcheck --enable=deadcode --enable=typecheck --enable=errcheck ./...

# coverage report
RUN mkdir -p target && /script/coverage.sh

# submit coverage to coverals if COVERALLS_TOKEN in env
RUN if [ -z "$COVERALLS_TOKEN" ] ; then \
    echo "coverall not enabled" ; \
    else goveralls -coverprofile=.cover/cover.out -service=travis-ci -repotoken $COVERALLS_TOKEN || echo "coverall failed!"; fi && \
    cat .cover/cover.out

RUN \
    if [ -z "$CI" ] ; then \
    echo "runs outside of CI" && version=$(/script/git-rev.sh); \
    else version=${GIT_BRANCH}-${GITHUB_SHA:0:7}-$(date +%Y%m%dT%H:%M:%S); fi && \
    echo "version=$version" && \
    go build -o feed-master -ldflags "-X main.revision=${version} -s -w" ./app && \
    ls -la /build/feed-master


FROM umputun/baseimage:app-latest

COPY --from=build /build/feed-master/feed-master /srv/feed-master
COPY app/webapp /srv/webapp
RUN \
    chown -R app:app /srv && \
    chmod +x /srv/feed-master

WORKDIR /srv

CMD ["/srv/feed-master"]
ENTRYPOINT ["/init.sh"]
