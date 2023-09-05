package dnstestserver

import (
	"sync"

	"github.com/miekg/dns"
)

type recordNameType struct {
	Name string
	Type uint16
}

type RRStore struct {
	mu sync.RWMutex
	m  map[recordNameType][]dns.RR
}

func (s *RRStore) Add(rr dns.RR) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.m == nil {
		s.m = map[recordNameType][]dns.RR{}
	}
	hdr := rr.Header()
	key := recordNameType{Name: dns.Fqdn(hdr.Name), Type: hdr.Rrtype}
	_, exists := s.m[key]
	s.m[key] = append(s.m[key], rr)
	return exists
}

func (s *RRStore) Find(name string, recordType uint16) []dns.RR {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.m[recordNameType{Name: dns.Fqdn(name), Type: recordType}]
}

func (s *RRStore) Remove(name string, recordType uint16) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := recordNameType{Name: dns.Fqdn(name), Type: recordType}
	_, ok := s.m[key]
	if ok {
		delete(s.m, key)
	}
	return ok
}
