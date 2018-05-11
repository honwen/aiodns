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
func NewAutoDNS(inside, outside func(dns.ResponseWriter, *dns.Msg), outsideListSuffix string) (ad *AutoDNS) {
	ad = &AutoDNS{
		insideHandleFunc:  inside,
		outsideHandleFunc: outside,
	}
	if 0 == len(outsideListSuffix) {
		ad.outsideListIndex = suffixarray.New([]byte(outsideList))
	} else {
		ad.outsideListIndex = suffixarray.New([]byte(outsideListSuffix))
	}

	return
}

// HandleFunc auto handles DNS request
func (ad *AutoDNS) HandleFunc(w dns.ResponseWriter, req *dns.Msg) {
	var err error
	/* any questions? */
	if len(req.Question) < 1 {
		return
	}

	rmsg := new(dns.Msg)
	rmsg.SetReply(req)
	q := req.Question[0]
	switch q.Qtype {
	case dns.TypeA:
		glog.V(LINFO).Infoln("requesting:", q.Name, dns.TypeToString[q.Qtype])

		for qName := q.Name[:len(q.Name)-1]; strings.Count(qName, `.`) > 0; qName = qName[strings.Index(qName, `.`)+1:] {
			offsets := ad.outsideListIndex.Lookup([]byte(qName), 1)
			if len(offsets) > 0 {
				glog.V(LDEBUG).Infoln(qName, "Hit OutsideList")
				ad.outsideHandleFunc(w, req)
				return
			}
		}
		ad.insideHandleFunc(w, req)
		return
	case dns.TypeANY:
		glog.V(LINFO).Infoln("request-block", q.Name, dns.TypeToString[q.Qtype])
	default:
		glog.V(LINFO).Infoln("requesting:", q.Name, dns.TypeToString[q.Qtype])
		ad.outsideHandleFunc(w, req)
		return
	}

	// fmt.Println(rmsg)
	if err = w.WriteMsg(rmsg); nil != err {
		glog.V(LINFO).Infoln("Response faild, rmsg:", err)
	}
}
