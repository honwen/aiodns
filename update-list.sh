#!/bin/bash

SRC=https://raw.githubusercontent.com/honwen/openwrt-dnsmasq-extra/master/dnsmasq-extra/files/data

# init_tldn.go
cat <<EOF >init_tldn.go
package main

const tldnList = \`
$(curl -sSL ${SRC}/tldn.gz | zcat | grep -v 'xn--')
\`
EOF

# init_spec.go
cat <<EOF >init_spec.go
package main

const specList = \`
$(curl -sSL ${SRC}/gfwlist.gz | zcat | sed '/[0-9]$/d' | grep -v 'xn--')
\`
EOF

# init_bypass.go
cat <<EOF >init_bypass.go
package main

const bypassList = \`
$(curl -sSL ${SRC}/direct.gz | zcat | sed '/^\./d' | grep -v 'xn--')
\`
EOF

# tidy-up
go mod tidy
TIDY_UP=1 go run . -V 2>/dev/null | sed -n 's+#tideSpec *++p' >/tmp/tideSpec
TIDY_UP=1 go run . -V 2>/dev/null | sed -n 's+#tideBypass *++p' >/tmp/tideBypass
which shadowsocks-helper >/dev/null 2>&1 && {
    shadowsocks-helper tide -i /tmp/tideSpec -o /tmp/tideSpec
    shadowsocks-helper tide -i /tmp/tideBypass -o /tmp/tideBypass
} || {
    sort -u /tmp/tideSpec -o /tmp/tideSpec
    sort -u /tmp/tideBypass -o /tmp/tideBypass
}

# Tidy: init_spec.go
cat <<EOF >init_spec.go
package main

const specList = \`
$(cat /tmp/tideSpec)
\`
EOF

# Tidy: init_bypass.go
cat <<EOF >init_bypass.go
package main

const bypassList = \`
$(cat /tmp/tideBypass)
\`
EOF

rm -f /tmp/tideSpec
rm -f /tmp/tideBypass

wc -l init_*.go
go fmt .
