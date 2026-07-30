# Vendored internal packages

Most packages in this directory are **verbatim copies** of the same-named
internal packages of [AdguardTeam/dnsproxy](https://github.com/AdguardTeam/dnsproxy/tree/v0.83.1/internal),
kept in sync mechanically by [`update.sh`](./update.sh):

- `dnsmsg`
- `middleware` (upstream of the former local `handler` package)
- `netutil` (only the files used by this project)

Do not edit these files by hand — changes will be overwritten on the next
sync.  `cmd` is the only **adapted fork** (exported `Configuration`/`RunProxy`,
`const.go`, fewer options) and is ported manually; `update.sh` prints the
upstream diff as a guide when it changes.

## Usage

```sh
# Sync with the latest dnsproxy release:
./internal/update.sh

# Sync with a specific release:
./internal/update.sh v0.83.1
```

The script mirrors the verbatim files, stamps `cmd/const.go`, runs
`go get`/`go mod tidy`, and verifies with `go build`/`go vet`/`gofmt`.
If a dnsproxy release changes APIs that the adapted `cmd` package relies
on, the build failure output includes the upstream `internal/cmd` diff to
port by hand.
