FROM golang:alpine as builder
RUN apk add --update git curl
RUN go get -u -v \
        -ldflags "-X main.version=$(curl -sSL https://api.github.com/repos/chenhw2/aiodns/commits/master | \
            sed -n '1,9{/sha/p; /date/p}' | sed 's/.* \"//g' | cut -c1-10 | tr [a-z] [A-Z] | sed 'N;s/\n/@/g')" \
        github.com/chenhw2/aiodns

FROM chenhw2/alpine:base
LABEL MAINTAINER CHENHW2 <https://github.com/chenhw2>

# /usr/bin/aiodns
COPY --from=builder /go/bin /usr/bin

USER nobody

ENV ARGS=

EXPOSE 5300
EXPOSE 5300/udp

CMD aiodns -T -U ${ARGS} --logtostderr -V 3
