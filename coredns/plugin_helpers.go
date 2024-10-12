package coredns

import (
	"context"
	"fmt"
	"log"

	"github.com/endreszabo/redns-apiserver/constants"
	"github.com/mediocregopher/radix/v4"
	"github.com/miekg/dns"
)

func UnpackRRwithQname(buf []byte, qname string) (dns.RR, error) {
	rr, _, err := dns.UnpackRR(buf, 0)

	if err != nil {
		return nil, err
	}

	//here we restore the original qname
	rr.Header().Name = qname
	return rr, nil
}

func GetRecords(ctx context.Context, server radix.Client, qname string, qtype string) ([]dns.RR, error) {
	var rv []dns.RR
	var key string
	s := (radix.ScannerConfig{Pattern: constants.ScanSpecificActiveKeysTemplate(qname, qtype)}).New(server)
	for s.Next(ctx, &key) {
		var redisRv string
		var redisTtl int

		mn := radix.Maybe{Rcv: &redisRv}

		if err := server.Do(ctx, radix.Cmd(&mn, "GET", key)); err != nil {
			return nil, fmt.Errorf("could not get redis value; key='%s', err='%v'", key, err.Error())
		} else if mn.Null {
			return nil, fmt.Errorf("redis does not have key; key='%s'", key)
		}

		rr, _, err := dns.UnpackRR([]byte(redisRv), 0)

		if err != nil {
			return nil, err
		}

		//here we restore the original qname
		rr.Header().Name = qname
		if true {
			log.Printf("got redis RR back: %#v", rr)
		}

		//clamp the RR TTL to the Redis key TTL (if set)
		if err := server.Do(ctx, radix.Cmd(&redisTtl, "TTL", key)); err != nil {
			return nil, fmt.Errorf("could not get key TTL value; key='%s', err='%v'", key, err.Error())
		}
		if redisTtl > -1 {
			TtlUint32 := uint32(redisTtl)
			if rr.Header().Ttl > TtlUint32 {
				rr.Header().Ttl = TtlUint32
			}
		}

		rv = append(rv, rr)
	}
	if err := s.Close(); err != nil {
		return nil, err
	}

	return rv, nil
}

func searchGlue(ctx context.Context, client radix.Client, qname string) ([]dns.RR, error) {
	var rv []dns.RR
	records, err := GetRecords(ctx, client, qname, "A")
	if err != nil {
		return nil, err
	}
	rv = append(rv, records...)

	records, err = GetRecords(ctx, client, qname, "AAAA")
	if err != nil {
		return nil, err
	}
	rv = append(rv, records...)
	return rv, nil
}

func HandleRequest(ctx context.Context, client radix.Client, r *dns.Msg) (*dns.Msg, error) {
	qname := dns.Name(r.Question[0].Name).String()
	qtype := r.Question[0].Qtype
	qtypeStr := dns.TypeToString[qtype]

	m := new(dns.Msg)
	answers, err := GetRecords(ctx, client, qname, qtypeStr)
	if err != nil {
		return nil, err
	}

	m.Answer = append(m.Answer, answers...)
	if qtype == dns.TypeNS {
		for _, answer := range m.Answer {
			glueRecords, err := searchGlue(ctx, client, answer.(*dns.NS).Ns)
			if err != nil {
				return nil, err
			}
			m.Extra = append(m.Extra, glueRecords...)
		}
	}

	if qtype == dns.TypeMX {
		for _, answer := range m.Answer {
			glueRecords, err := searchGlue(ctx, client, answer.(*dns.MX).Mx)
			if err != nil {
				return nil, err
			}
			m.Extra = append(m.Extra, glueRecords...)
		}
	}

	if qtype == dns.TypeSRV {
		for _, answer := range m.Answer {
			glueRecords, err := searchGlue(ctx, client, answer.(*dns.SRV).Target)
			if err != nil {
				return nil, err
			}
			m.Extra = append(m.Extra, glueRecords...)
		}
	}

	m.SetReply(r)
	m.Authoritative = true
	log.Printf("%s\n%s\n", m.String(), m.Question[0].Name)
	return m, nil
}
