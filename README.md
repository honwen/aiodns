# AIO DNS

A All-In-One DNS Solution, A Specail-List Rule Generator of `AdguardTeam/dnsproxy`

### Source

- https://github.com/honwen/aiodns

### Thanks

- https://github.com/AdguardTeam/dnsproxy
- https://github.com/honwen/openwrt-dnsmasq-extra

### Docker

- https://hub.docker.com/r/chenhw2/aiodns

### Usage

```bash
$ docker pull chenhw2/aiodns

$ docker run -d \
    ---network=host \
    -e PORT=53 \
    chenhw2/aiodns
```

### Help

```bash
$ docker run --rm chenhw2/aiodns -h
NAME:
   AIO DNS - All In One Clean DNS Solution.

USAGE:
   aiodns [global options] command [command options] [arguments...]

VERSION:
   Git:[MISSING BUILD VERSION [GIT HASH]] (go version)

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --listen value, -l value            Listening address (default: ":5300")
   --upstream value, -u value          An upstream to be default used (can be specified multiple times) (default: "tls://dns.pub", ...)
   --special-upstream value, -U value  An upstream to be special used (can be specified multiple times) (default: "https://8.8.8.8/dns-query", ...)
   --fallback value, -f value          Fallback resolvers to use when regular ones are unavailable, can be specified multiple times (default: "tcp://9.9.9.11:9953", ...)
   --bootstrap value, -b value         Bootstrap DNS for DoH and DoT, can be specified multiple times (default: "tls://223.5.5.5", ...)
   --special-list value, -L value      List of domains using special-upstream (can be specified multiple times)
   --bypass-list value, -B value       List of domains bypass special-upstream (can be specified multiple times)
   --edns value, -e value              Send EDNS Client Address to default upstreams
   --timeout value, -t value           Timeout of Each upstream, [1, 59] seconds (default: 3)
   --cache, -C                         If specified, DNS cache is enabled
   --insecure, -I                      If specified, disable SSL/TLS Certificate check (for some OS without ca-certificates)
   --ipv6-disabled, -R                 If specified, all AAAA requests will be replied with NoError RCode and empty answer
   --refuse-any, -A                    If specified, refuse ANY requests
   --fastest-addr, -F                  If specified, Respond to A or AAAA requests only with the fastest IP address
   --verbose, -V                       If specified, Verbose output
   --help, -h                          show help
   --version, -v                       print the version
```
