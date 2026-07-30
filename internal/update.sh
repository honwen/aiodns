#!/usr/bin/env bash
#
# update.sh syncs the vendored internal packages with the upstream
# AdguardTeam/dnsproxy module and bumps the dependency in go.mod.
#
# Usage:
#
#	./internal/update.sh [version]
#
# version is a dnsproxy release like "v0.83.1" (the leading "v" may be
# omitted).  If unset, the latest release is resolved from the Go module
# proxy.
#
# The sync policy (see internal/README.md):
#
#   - internal/dnsmsg, internal/middleware, internal/netutil are VERBATIM
#     copies of the same-named upstream internal packages (only the files
#     listed in SYNC_FILES below).  They are overwritten mechanically.
#   - internal/cmd is an ADAPTED fork (exported Configuration/RunProxy,
#     const.go, fewer options) and is NOT overwritten.  When upstream
#     changes it, the script prints the upstream diff as a porting guide
#     and lets the build check decide whether manual work is needed.
#
# shellcheck disable=SC2015
set -euo pipefail

# cd to the repository root (the parent of this script's directory).
cd "$(dirname "$0")/.."

readonly MODULE='github.com/AdguardTeam/dnsproxy'
readonly PROXY_BASE='https://proxy.golang.org/github.com/!adguard!team/dnsproxy'

# SYNC_FILES lists the files mirrored verbatim from the upstream module.
readonly SYNC_FILES='
	internal/dnsmsg/constructor.go
	internal/middleware/constructor.go
	internal/middleware/hosts.go
	internal/middleware/ipv6halt.go
	internal/middleware/middleware.go
	internal/netutil/netutil.go
	internal/netutil/paths.go
	internal/netutil/paths_unix.go
	internal/netutil/paths_windows.go
'

log() { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m!!!\033[0m %s\n' "$*" >&2; }
die() {
	printf '\033[1;31m***\033[0m %s\n' "$*" >&2
	exit 1
}

# mod_dir prints the local module cache directory of $MODULE at version $1,
# downloading it if necessary.
mod_dir() {
	local ver="$1"
	local dir
	dir="$(go list -m -f '{{.Dir}}' "${MODULE}@${ver}" 2>/dev/null)" \
		|| die "cannot resolve ${MODULE}@${ver}"
	[[ -d $dir ]] || die "module directory not found for ${MODULE}@${ver}"

	printf '%s\n' "$dir"
}

# latest_version queries the Go module proxy for the latest release version.
latest_version() {
	curl -sSf --max-time 30 "${PROXY_BASE}/@latest" \
		| sed -n 's/.*"Version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p'
}

old_ver="$(go list -m -f '{{.Version}}' "$MODULE")" \
	|| die "cannot read current $MODULE version from go.mod"

new_ver="${1:-}"
if [[ -z $new_ver ]]; then
	log "resolving latest ${MODULE} version from proxy.golang.org"
	new_ver="$(latest_version)" || die "cannot resolve latest version; pass it explicitly"
	[[ -n $new_ver ]] || die "cannot resolve latest version; pass it explicitly"
fi
[[ $new_ver == v* ]] || new_ver="v${new_ver}"

log "current: ${old_ver}  target: ${new_ver}"
if [[ $old_ver == "$new_ver" ]]; then
	log "already at ${new_ver}; re-syncing files anyway"
fi

new_dir="$(mod_dir "$new_ver")"
log "upstream source: ${new_dir}"

# 1. Mirror the verbatim file set.
for f in $SYNC_FILES; do
	[[ -f ${new_dir}/${f} ]] || die "upstream file missing: ${f} (update SYNC_FILES)"
	mkdir -p "$(dirname "$f")"
	cp "${new_dir}/${f}" "$f"
	# Module cache files are read-only.
	chmod u+w "$f"
	# Rewrite cross-internal imports if upstream ever adds them.
	if grep -q 'github.com/AdguardTeam/dnsproxy/internal/' "$f"; then
		sed -i 's|github.com/AdguardTeam/dnsproxy/internal/|github.com/honwen/aiodns/internal/|g' "$f"
		warn "rewrote internal imports in ${f}; verify the result"
	fi
done
log "synced $(echo "$SYNC_FILES" | wc -w) verbatim files"

# 2. Stamp the tracked upstream version.
sed -i "s/^\(	Version = \)\"[^\"]*\"/\1\"${new_ver}\"/" internal/cmd/const.go
grep -q "\"${new_ver}\"" internal/cmd/const.go \
	|| die "failed to stamp version into internal/cmd/const.go"
sed -i "s|dnsproxy/blob/[^/]*/internal|dnsproxy/blob/${new_ver}/internal|" internal/README.md

# 3. Bump the dependency and tidy.
log "go get ${MODULE}@${new_ver}"
go get "${MODULE}@${new_ver}"
go mod tidy

# 4. Verify.  On failure, print the upstream changes in the adapted
#    internal/cmd package as a porting guide.
if ! go build ./...; then
	warn "build broken after upgrade to ${new_ver}!"
	warn "internal/cmd is an adapted fork; upstream changes since ${old_ver}:"
	old_dir="$(mod_dir "$old_ver")"
	diff -ru "${old_dir}/internal/cmd" "${new_dir}/internal/cmd" || true
	die  "port the upstream changes above into internal/cmd, then re-run the build"
fi
go vet ./...
gofmt -l . | grep . && die "gofmt issues found"
log "build, vet, gofmt: OK"

# 5. FYI: upstream changes in the adapted package (may contain optional
#    features worth porting).  Note that diff exits 1 on differences, so it
#    must stay in an if condition to survive "set -e".
if [[ $old_ver != "$new_ver" ]]; then
	old_dir="$(mod_dir "$old_ver")"
	if ! diff -rq "${old_dir}/internal/cmd" "${new_dir}/internal/cmd" >/dev/null 2>&1; then
		log "upstream internal/cmd changed ${old_ver} -> ${new_ver};"
		log "review with:  diff -ru '${old_dir}/internal/cmd' '${new_dir}/internal/cmd'"
	fi
fi

# 6. Remind about the Docker base image if the go directive moved.
if git diff go.mod | grep -q '^+go '; then
	go_ver="$(sed -n 's/^go \([0-9]*\.[0-9]*\).*/\1/p' go.mod)"
	warn "go directive changed; check 'FROM golang:${go_ver}' in Dockerfile"
fi

log "done: ${MODULE} ${old_ver} -> ${new_ver}"
