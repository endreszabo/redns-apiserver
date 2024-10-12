package redns_rpc

import (
	"fmt"

	"github.com/endreszabo/redns-apiserver/constants"
	"github.com/endreszabo/redns-apiserver/coredns"
	"github.com/miekg/dns"
)

type ExtendedStrayRecord struct {
	*DetailedStrayEntity
	RedisKey *string
	RR       *dns.RR
}

func (r *ExtendedStrayRecord) DecodeBufToRR(buf string) error {
	rr, err := coredns.UnpackRRwithQname([]byte(buf), r.QName)
	if err != nil {
		return err
	}
	r.RR = &rr
	return nil
}

func (r *ExtendedStrayRecord) ToWireFormat(keepQname bool) ([]byte, error) {
	buf := make([]byte, 1024)
	if r.RR == nil {
		return buf, fmt.Errorf("RR is nil")
	}
	rr := *r.RR
	if rr.Header().Ttl == 0 {
		rr.Header().Ttl = 300
	}
	if !keepQname {
		// we can compact the record size as
		// CoreDNS plugin does not need this, also redundant
		rr.Header().Name = "."
	}

	off, err := dns.PackRR(rr, buf, 0, nil, false)
	if err != nil {
		return buf, err
	}
	return buf[:off], nil
}

func (r *DetailedStrayEntity) ToRfc1035() string {
	if r.Rfc1035 == "" {
		return r.QName + "\t" + r.QType + "\t(missing)"
	}
	return r.Rfc1035
}

// return redis Key for this Record
func (r *DetailedStrayEntity) Key() string {
	return constants.StrayEntityKeyTemplate(r.QName, r.QType, r.Id)
}
