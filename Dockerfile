FROM umputun/baseimage:buildgo-latest as build

ARG COVERALLS_TOKEN
ARG CI
ARG TRAVIS
ARG TRAVIS_BRANCH
ARG TRAVIS_COMMIT
ARG TRAVIS_JOB_ID
ARG TRAVIS_JOB_NUMBER
ARG TRAVIS_OS_NAME
ARG TRAVIS_PULL_REQUEST
ARG TRAVIS_PULL_REQUEST_SHA
ARG TRAVIS_REPO_SLUG
ARG TRAVIS_TAG
ARG TRAVIS_BUILD_NUMBER

WORKDIR /go/src/github.com/umputun/feed-master
ADD . /go/src/github.com/umputun/feed-master

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
    if [ -z "$TRAVIS" ] ; then \
    echo "runs outside of travis" && version=$(/script/git-rev.sh); \
    else version=${TRAVIS_BRANCH}-${TRAVIS_BUILD_NUMBER}-${TRAVIS_COMMIT:0:7}-$(date +%Y%m%dT%H:%M:%S); fi && \
    echo "version=$version" && \
    go build -o feed-master -ldflags "-X main.revision=${version} -s -w" ./app


FROM umputun/baseimage:app-latest

COPY --from=build /go/src/github.com/umputun/feed-master/feed-master /srv/feed-master
RUN \
    chown -R app:app /srv && \
    chmod +x /srv/feed-master

WORKDIR /srv

CMD ["/srv/feed-master"]
ENTRYPOINT ["/init.sh"]
