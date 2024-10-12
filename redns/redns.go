package redns

import (
	"context"
	"fmt"
	"strconv"

	"github.com/endreszabo/redns-apiserver/constants"
	pb "github.com/endreszabo/redns-apiserver/proto"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mediocregopher/radix/v4"
)

type StrayServer struct {
	Log    zerolog.Logger
	Server radix.Client
}

func (s *StrayServer) Close() error {
	if s.Server != nil {
		return s.Server.Close()
	}
	return nil
}

func (s *StrayServer) String() string {
	return s.Server.Addr().String()
}

func (s *StrayServer) GetStrayRecords(ctx context.Context, template string, withValues bool) ([]*pb.ExtendedStrayRecord, error) {
	var rv []*pb.ExtendedStrayRecord
	var key string

	s.Log.Debug().Str("template", template).Msg("starting to collect keys for server")
	scanner := (radix.ScannerConfig{Pattern: template}).New(s.Server)
	for scanner.Next(ctx, &key) {
		dr, err := NewDetailedStrayEntityFromK(key)
		if err != nil {
			return nil, err
		}
		r := &pb.ExtendedStrayRecord{
			DetailedStrayEntity: dr,
			RedisKey:            &key,
		}

		var bufStr string
		err = s.Server.Do(ctx, radix.Cmd(&bufStr, "TTL", key))
		if err != nil {
			return nil, err
		}

		//set expiry of redis key
		uintExpiry, err := strconv.ParseInt(bufStr, 10, 32)
		if err != nil {
			return nil, err
		}
		if uintExpiry == -1 {
			r.Expiry = 0
		} else if uintExpiry < 0 {
			return nil, fmt.Errorf("expiry value is negative: %d", uintExpiry)
		} else {
			r.Expiry = uint32(uintExpiry)
		}

		//caller wants RRs
		if withValues {
			err := s.Server.Do(ctx, radix.Cmd(&bufStr, "GET", key))
			if err != nil {
				return nil, err
			}
			err = r.DecodeBufToRR(bufStr)
			if err != nil {
				return nil, err
			}
			r.Rfc1035 = (*r.RR).String()
		}

		s.Log.Debug().Str("key", key).Msg("collected key for server")
		rv = append(rv, r)
	}
	if err := scanner.Close(); err != nil {
		return nil, err
	}

	return rv, nil
}

/*
func (r *genericRecordData) ToWireFormat(keepQname bool) ([]byte, error) {
	buf := make([]byte, 1024)
	if r.rr == nil {
		return buf, fmt.Errorf("RR is nil")
	}
	rr := *r.rr
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
*/

// deprecated
/*
func getRednsKeys(ctx context.Context, debug bool, redisServers redisServers, keyTemplate string) (*[]string, error) {
	perServerKeys := make(map[string][]string)
	var firstServerKeys []string
	var key string

	for _, server := range redisServers.server {
		serverAddr := server.Addr().String()
		if debug {
			log.Printf("starting to collect keys for server; template='%s'; server='%s'\n", keyTemplate, serverAddr)
		}
		s := (radix.ScannerConfig{Pattern: keyTemplate}).New(server)
		for s.Next(ctx, &key) {
			perServerKeys[serverAddr] = append(perServerKeys[serverAddr], key)
			if debug {
				log.Printf("collected key for server; key='%s'; server='%s'\n", key, serverAddr)
			}
		}
		if err := s.Close(); err != nil {
			return nil, err
		}
	}

	//compare the results
	for idx, server := range redisServers.server {
		serverAddr := server.Addr().String()
		var firstServerAddr string
		if idx == 0 {
			firstServerKeys = perServerKeys[serverAddr]
			firstServerAddr = serverAddr
		} else {
			if !Equal(firstServerKeys, perServerKeys[serverAddr]) {
				return nil, fmt.Errorf("arrays of active keys are not equal on servers; server_a='%s', server_b='%s'", firstServerAddr, serverAddr)
			}
		}
	}

	log.Printf("%#v", perServerKeys)
	return &firstServerKeys, nil
}
*/

/*
func NewRednsRecordFromK(key string) (*rednsRecord, error) {
	splitKey := strings.SplitN(key, "/", 6)
	r := rednsRecord{
		genericRecordData: genericRecordData{
			Qname:    splitKey[3],
			Qtype:    splitKey[4],
			redisKey: &key,
		},
		Status: splitKey[5],
		Id:     splitKey[6],
	}
	if strings.Index(r.Id, constants.RednsSubsystemName+"/") == 0 {
		return nil, status.Errorf(codes.Internal, "key from managed keyspace was received: %v", r.Id)
	}
	return &r, nil
}

func NewRednsRecordFromKV(key string, value string) (*rednsRecord, error) {
	r, err := NewRednsRecordFromK(key)
	if err != nil {
		return nil, err
	}
	rr, err := coredns.UnpackRRwithQname([]byte(value), r.Qname)
	r.rr = &rr
	if err != nil {
		return nil, err
	}
	r.Rfc1035 = rr.String()
	return r, nil
}

func NewRednsRecordFromRR(RR dns.RR, qnameOverride string) (*rednsRecord, error) {
	rv := rednsRecord{
		genericRecordData: genericRecordData{
			Qname:   RR.Header().Name,
			Qtype:   dns.TypeToString[RR.Header().Rrtype],
			Rfc1035: RR.String(),
			rr:      &RR,
		},
	}
	if qnameOverride != "" {
		rv.Qname = qnameOverride
	}
	return &rv, nil
}

func NewRednsRecordFromRfc1035String(rfc1035value string) (*rednsRecord, error) {
	rr, err := dns.NewRR(rfc1035value)
	if err != nil {
		return nil, err
	}
	return NewRednsRecordFromRR(rr, "")
}

*/

func (s *StrayServer) DeleteStrayById(ctx context.Context, entity *pb.StrayEntity) error {
	key := constants.StrayEntityKeyTemplate(entity.QName, entity.QType, entity.Id)
	var delRetval string
	s.Server.Do(ctx, radix.Cmd(&delRetval, "DEL", key))
	if delRetval != "OK" {
		return fmt.Errorf("DEL call returned an unexpected answer: %q", delRetval)
	}
	return nil
}

func (s *StrayServer) DeleteStrayByValue(ctx context.Context, entity *pb.ExtendedStrayRecord) (int, error) {
	var rv = 0

	records, err := s.GetStrayRecords(ctx, constants.StrayEntityKeyTemplate(entity.QName, entity.QType, "*"), true)
	if err != nil {
		return rv, err
	}
	for _, record := range records {
		if record.Rfc1035 == entity.Rfc1035 {
			var delRetval string
			s.Server.Do(ctx, radix.Cmd(&delRetval, "DEL", *record.RedisKey))
			if delRetval != "OK" {
				return rv, fmt.Errorf("DEL call returned an unexpected answer: %q", delRetval)
			}
			rv++
		}
	}

	return rv, nil
}

/* foobar */
func (s *StrayServer) DeleteStrayByRfc1035String(ctx context.Context, entity *pb.StrayByValueRequest) (int, error) {
	req, err := NewStrayRecordFromRfc1035String(entity.Rfc1035)
	if err != nil {
		return 0, err
	}
	return s.DeleteStrayByValue(ctx, req)
}

func (s *StrayServer) AddStray(ctx context.Context, req *pb.StrayModifyRequest) error {
	r, err := NewStrayRecordFromRfc1035String(req.Rfc1035)
	if err != nil {
		return err
	}
	r.Id = req.Id
	buf, err := r.ToWireFormat(false)
	if err != nil {
		return err
	}
	keyName := r.Key()

	if !req.AllowOverwrite {
		var recordExists string
		err = s.Server.Do(ctx, radix.Cmd(&recordExists, "EXISTS", keyName))
		if err != nil {
			return err
		}
		if recordExists == "1" {
			return status.Error(codes.InvalidArgument, "same record with this ID already exists")
		}
	}
	var retval string
	if err := s.Server.Do(ctx, radix.Cmd(&retval, "SET", keyName, string(buf))); err != nil {
		return err
	}
	if retval != "OK" {
		return fmt.Errorf("SET call returned an unexpected answer: %q", retval)
	}

	if req.Expiry != nil && *req.Expiry > 0 {
		if err := s.Server.Do(ctx, radix.Cmd(&retval, "EXPIRE", keyName, fmt.Sprint(*req.Expiry))); err != nil {
			return err
		}
		if retval != "1" {
			return status.Errorf(codes.Internal, "EXPIRE failed on server: %v", retval)
		}
		//	r.Expiry = *req.Expiry
	}

	return nil
}
