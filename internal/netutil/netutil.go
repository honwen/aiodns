// https://github.com/AdguardTeam/dnsproxy/blob/v0.61.0/internal/netutil/netutil.go#L51
package netutil

import (
	"net/netip"
	"strings"
)

// TODO(e.burkov):  Move to golibs.
func ParseSubnet(s string) (p netip.Prefix, err error) {
	if strings.Contains(s, "/") {
		p, err = netip.ParsePrefix(s)
		if err != nil {
			return netip.Prefix{}, err
		}
	} else {
		var ip netip.Addr
		ip, err = netip.ParseAddr(s)
		if err != nil {
			return netip.Prefix{}, err
		}

		p = netip.PrefixFrom(ip, ip.BitLen())
	}

	return p, nil
}
