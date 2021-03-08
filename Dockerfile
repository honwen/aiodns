FROM golang:alpine as builder
ENV CGO_ENABLED=0 \
    GO111MODULE=on
RUN apk add --update git curl
ADD . $GOPATH/src/github.com/honwen/aiodns
RUN set -ex \
    && cd $GOPATH/src/github.com/honwen/aiodns \
    && go build -ldflags "-X main.version=$(curl -sSL https://api.github.com/repos/honwen/aiodns/commits/master | \
            sed -n '{/sha/p; /date/p;}' | sed 's/.* \"//g' | cut -c1-10 | tr '[:lower:]' '[:upper:]' | sed 'N;s/\n/@/g' | head -1)" . \
    && mv aiodns $GOPATH/bin/ \
    && aiodns -v

FROM chenhw2/alpine:base
LABEL MAINTAINER honwen <https://github.com/honwen>

# /usr/bin/aiodns
COPY --from=builder /go/bin /usr/bin

USER nobody

ENV ARGS=

EXPOSE 5300
EXPOSE 5300/udp

CMD aiodns -T -U ${ARGS} --logtostderr -V 3
