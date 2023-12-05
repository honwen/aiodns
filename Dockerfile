FROM golang:1.21 as builder

WORKDIR /workdir

ADD . ./

RUN set -ex \
    && GOPROXY='https://mirrors.cloud.tencent.com/go/,direct' go mod download -x \
    && go generate -x \
    && CGO_ENABLED=0 go build -v -ldflags "-X main.VersionString=$(curl -sSL https://api.github.com/repos/honwen/aiodns/commits/master | \
            sed -n '{/sha/p; /date/p;}' | sed 's/.* \"//g' | cut -c1-10 | tr '[:lower:]' '[:upper:]' | sed 'N;s/\n/@/g' | head -1)" . \
    && ./aiodns -v

FROM chenhw2/alpine:base
LABEL MAINTAINER honwen <https://github.com/honwen>

# /usr/bin/aiodns
COPY --from=builder /workdir/aiodns /usr/bin

USER nobody

ENV PORT=5300 \
    ARGS="-C -F -A -R -V -L=https://raw.githubusercontents.com/honwen/openwrt-dnsmasq-extra/master/dnsmasq-extra/files/data/gfwlist -L=https://raw.githubusercontents.com/honwen/openwrt-dnsmasq-extra/master/dnsmasq-extra/files/data/tldn -L=https://raw.githubusercontents.com/Loyalsoldier/v2ray-rules-dat/release/greatfire.txt"

CMD aiodns -l=:${PORT} ${ARGS}
