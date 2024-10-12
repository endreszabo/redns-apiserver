package redns

import (
	"strings"

	"github.com/endreszabo/redns-apiserver/coredns"
	pb "github.com/endreszabo/redns-apiserver/proto"
	"github.com/miekg/dns"
	"google.golang.org/grpc/metadata"
)

func getStringFromMetadata(md metadata.MD, key string) string {
	rv := md.Get(key)
	if len(rv) < 1 {
		return ""
	}
	return rv[0]
}

func NewDetailedStrayEntityFromK(key string) (*pb.DetailedStrayEntity, error) {
	splitKey := strings.SplitN(key, "/", 8)
	r := pb.DetailedStrayEntity{
		QName: splitKey[3],
		QType: splitKey[4],
		Id:    splitKey[7],
	}
	return &r, nil
}

func NewDetailedStrayEntityFromKV(key string, value string) (*pb.DetailedStrayEntity, error) {
	r, err := NewDetailedStrayEntityFromK(key)
	if err != nil {
		return nil, err
	}
	rr, err := coredns.UnpackRRwithQname([]byte(value), r.QName)
	if err != nil {
		return nil, err
	}
	r.Rfc1035 = rr.String()
	return r, nil
}

// taken from: https://yourbasic.org/golang/compare-slices/
// Equal tells whether a and b contain the same elements.
// A nil argument is equivalent to an empty slice.
func Equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func GenRRBuf(rr dns.RR, keepQname bool) ([]byte, error) {
	if rr.Header().Ttl == 0 {
		rr.Header().Ttl = 300
	}
	if !keepQname {
		rr.Header().Name = "."
	}
	buf := make([]byte, 1024)

	// we can compact the record size as
	// CoreDNS plugin does not need this, also redundant
	rr.Header().Name = "."
	off, err := dns.PackRR(rr, buf, 0, nil, false)
	if err != nil {
		return buf, err
	}
	return buf[:off], nil
}

func NewStrayRecordFromRR(RR *dns.RR, qnameOverride string) (*pb.ExtendedStrayRecord, error) {
	rv := pb.ExtendedStrayRecord{
		DetailedStrayEntity: &pb.DetailedStrayEntity{
			QName:   (*RR).Header().Name,
			QType:   dns.TypeToString[(*RR).Header().Rrtype],
			Rfc1035: (*RR).String(),
		},
		RedisKey: nil,
		RR:       RR,
	}
	if qnameOverride != "" {
		rv.QName = qnameOverride
	}
	return &rv, nil
}

func NewStrayRecordFromRfc1035String(rfc1035value string) (*pb.ExtendedStrayRecord, error) {
	rr, err := dns.NewRR(rfc1035value)
	if err != nil {
		return nil, err
	}
	return NewStrayRecordFromRR(&rr, "")
}
