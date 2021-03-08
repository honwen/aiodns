package main

import (
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/golang/glog"
	"github.com/miekg/dns"
)

const (
	minDNSPacketSize = 12 + 5

	ednsCSDefaultNetmaskV4 = 24  // default network mask for IPv4 address for EDNS ClientSubnet option
	ednsCSDefaultNetmaskV6 = 112 // default network mask for IPv6 address for EDNS ClientSubnet option

	scope = 24
)

// HandlerOptions specifies options to be used when instantiating a handler
type HandlerOptions struct {
	blockANY  bool
	blockAAAA bool
	edns      string // TODO
}

// Handler represents a DNS handler
type Handler struct {
	options   *HandlerOptions
	providers []upstream.Upstream
}

// NewHandler creates a new Handler
func NewHandler(upstream []upstream.Upstream, options HandlerOptions) *Handler {
	return &Handler{&options, upstream}
}

// HandleFunc handles a DNS request
func (h *Handler) HandleFunc(w dns.ResponseWriter, r *dns.Msg) {
	q := append([]dns.Question(nil), r.Question...)
	r.Question = []dns.Question{}

	for i := range q {
		switch q[i].Qtype {
		case dns.TypeANY:
			if h.options.blockANY {
				glog.V(LINFO).Infoln("request-Blocked", q[i].Name, dns.TypeToString[q[i].Qtype])
			} else {
				glog.V(LINFO).Infoln("requesting", q[i].Name, dns.TypeToString[q[i].Qtype])
				r.Question = append(r.Question, q[i])
			}
		case dns.TypeAAAA:
			if h.options.blockAAAA {
				glog.V(LINFO).Infoln("request-Blocked", q[i].Name, dns.TypeToString[q[i].Qtype])
			} else {
				glog.V(LINFO).Infoln("requesting", q[i].Name, dns.TypeToString[q[i].Qtype])
				r.Question = append(r.Question, q[i])
			}
		default:
			glog.V(LINFO).Infoln("requesting", q[i].Name, dns.TypeToString[q[i].Qtype])
			r.Question = append(r.Question, q[i])
		}
	}

	if len(r.Question) < 1 {
		r.Question = q
		resp := proxy.GenEmptyMessage(r, dns.RcodeSuccess, 60)
		// Write the response
		if err := w.WriteMsg(resp); err != nil {
			glog.V(LERROR).Infoln("provider failed", err)
		}
		return
	}

	// if len(h.options.edns) > 1 {
	// 	ednsIP := net.ParseIP(h.options.edns)
	// 	if ednsIP == nil {
	// 		glog.V(LWARNING).Infoln("cannot parse %s", h.options.edns)
	// 	} else {
	// 		e := new(dns.EDNS0_SUBNET)
	// 		e.Code = dns.EDNS0SUBNET
	// 		if ednsIP.To4() != nil {
	// 			e.Family = 1
	// 			e.SourceNetmask = ednsCSDefaultNetmaskV4
	// 			e.Address = ednsIP.To4().Mask(net.CIDRMask(int(e.SourceNetmask), 32))
	// 		} else {
	// 			e.Family = 2
	// 			e.SourceNetmask = ednsCSDefaultNetmaskV6
	// 			e.Address = ednsIP.Mask(net.CIDRMask(int(e.SourceNetmask), 128))
	// 		}
	// 		e.SourceScope = scope
	// 	}
	// }

	resp, upstream, err := upstream.ExchangeParallel(h.providers, r)
	glog.V(LINFO).Infoln("requested", q[0].Name, dns.TypeToString[q[0].Qtype], "[ using", upstream.Address(), "]")
	// Write the response
	if err = w.WriteMsg(resp); err != nil {
		glog.V(LERROR).Infoln("provider failed", err)
	}
	return
}
