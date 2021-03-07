### Source
- https://github.com/honwen/aiodns
  
### Thanks
- https://github.com/fardog/secureoperator
- https://github.com/shadowsocks/go-shadowsocks2
- https://developers.cloudflare.com/1.1.1.1/dns-over-https/
- https://developers.google.com/speed/public-dns/docs/dns-over-https
  
### Docker
- https://hub.docker.com/r/honwen/aiodns
  
### TODO
- Currently only Block DNS TYPE:```ANY```
- More thorough tests should be written
- No caching is implemented, and probably never will
  
### Usage
```
$ docker pull honwen/aiodns

$ docker run -d \
    -p "5300:5300/udp" \
    -p "5300:5300/tcp" \
    honwen/aiodns

```
### Help
```
$ docker run --rm honwen/aiodns -h
NAME:
   AIO DNS - All In One Clean DNS Solution.

USAGE:
   aiodns [global options] command [command options] [arguments...]

VERSION:
   Git:[MISSING BUILD VERSION [GIT HASH]] (go version)

COMMANDS:
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --listen value, -l value  Serve address (default: ":5300")
   --insecure, -I            Disable SSL/TLS Certificate check (for some OS without ca-certificates)
   --udp, -U                 Listen on UDP
   --tcp, -T                 Listen on TCP
   -V value                  log level for V logs (default: 2)
   --logtostderr             log to standard error instead of files
   --help, -h                show help
   --version, -v             print the version

```
