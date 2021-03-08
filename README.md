### Source
- https://github.com/honwen/aiodns
  
### Thanks
- https://github.com/AdguardTeam/dnsproxy
  
### Docker
- https://hub.docker.com/r/chenhw2/aiodns
  
### TODO
- Currently only Block DNS TYPE:```ANY```
- More thorough tests should be written
- No caching is implemented, and probably never will
  
### Usage
```
$ docker pull chenhw2/aiodns

$ docker run -d \
    -p "5300:5300/udp" \
    -p "5300:5300/tcp" \
    chenhw2/aiodns

```
### Help
```
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
   --listen value, -l value     Serve address (default: ":5300")
   --ou value, -o value         Outside Upstreams (default: "tls://dns.google", "tls://162.159.36.1", "tls://dns.adguard.com", "https://dns.google/dns-query", "https://doh.dns.sb/dns-query", "https://cloudflare-dns.com/dns-query")
   --iu value, -i value         Inside Upstreams (default: "tls://dns.pub", "tls://223.6.6.6", "https://doh.pub/dns-query", "https://dns.alidns.com/dns-query")
   --bootstrap value, -b value  Bootstrap Upstreams (default: "tls://223.5.5.5", "tls://1.0.0.1", "114.114.115.115")
   --insecure, -I               If specified, disable SSL/TLS Certificate check (for some OS without ca-certificates)
   --ipv6-disabled              If specified, all AAAA requests will be replied with NoError RCode and empty answer
   --refuse-any                 If specified, refuse ANY requests
   --udp, -U                    Listen on UDP
   --tcp, -T                    Listen on TCP
   -V value                     log level for V logs (default: 2)
   --logtostderr                log to standard error instead of files
   --help, -h                   show help
   --version, -v                print the version
```
