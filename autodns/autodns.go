package autodns

import (
	"index/suffixarray"
	"strings"

	"github.com/golang/glog"
	"github.com/miekg/dns"
)

// Log level for glog
const (
	LFATAL = iota
	LERROR
	LWARNING
	LINFO
	LDEBUG
)

// AutoDNS is a AutoDNS DNS-over-HTTP provider;
type AutoDNS struct {
	insideHandleFunc  func(dns.ResponseWriter, *dns.Msg)
	outsideHandleFunc func(dns.ResponseWriter, *dns.Msg)
	outsideListIndex  *suffixarray.Index
}

// NewAutoDNS creates a AutoDNS
func NewAutoDNS(inside, outside func(dns.ResponseWriter, *dns.Msg), outsideListSuffix string) (autod *AutoDNS) {
	autod = &AutoDNS{
		insideHandleFunc:  inside,
		outsideHandleFunc: outside,
	}
	if 0 == len(outsideListSuffix) {
		autod.outsideListIndex = suffixarray.New([]byte(outsideList))
	} else {
		autod.outsideListIndex = suffixarray.New([]byte(outsideListSuffix))
	}
	return
}

// HandleFunc auto handles DNS request
func (autod *AutoDNS) HandleFunc(w dns.ResponseWriter, req *dns.Msg) {
	/* any questions? */
	if len(req.Question) < 1 {
		return
	}

	rmsg := new(dns.Msg)
	rmsg.SetReply(req)
	q := req.Question[0]
	switch q.Qtype {
	case dns.TypeA, dns.TypeAAAA:
		glog.V(LINFO).Infoln("requesting:", q.Name, dns.TypeToString[q.Qtype])

		for qName := q.Name[:len(q.Name)-1]; strings.Count(qName, `.`) > 0; qName = qName[strings.Index(qName, `.`)+1:] {
			offsets := autod.outsideListIndex.Lookup([]byte(qName), 1)
			if len(offsets) > 0 {
				glog.V(LDEBUG).Infoln(qName, "Hit OutsideList")
				autod.outsideHandleFunc(w, req)
				return
			}
		}
		autod.insideHandleFunc(w, req)
		return
	default:
		glog.V(LINFO).Infoln("requesting:", q.Name, dns.TypeToString[q.Qtype])
		autod.outsideHandleFunc(w, req)
		return
	}
}
