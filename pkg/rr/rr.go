package rr

import (
	"fmt"
	"net"

	"github.com/miekg/dns"
)

type InvalidArgErr struct {
	msg string
}

func (e InvalidArgErr) Error() string {
	return e.msg
}

func InvalidArg(formatString string, args ...interface{}) InvalidArgErr {
	return InvalidArgErr{fmt.Sprintf(formatString, args...)}
}

func IsInvalidArg(err error) bool {
	_, ok := err.(InvalidArgErr)
	return ok
}

func New(typ uint16, name, value string) (dns.RR, error) {
	switch typ {
	case dns.TypeA:
		ip := net.ParseIP(value)
		if ip == nil {
			return nil, InvalidArg("invalid IP address: %s", value)
		}
		return NewA(name, ip), nil
	case dns.TypeCNAME:
		return NewCNAME(name, value), nil
	default:
		return nil, InvalidArg("unsupported RR type: %d", typ)
	}
}

func NewA(name string, ip net.IP) *dns.A {
	return &dns.A{
		Hdr: dns.RR_Header{
			Name:   dns.Fqdn(name),
			Rrtype: dns.TypeA,
			Class:  dns.ClassINET,
		},
		A: ip.To4(),
	}
}

func NewCNAME(name, value string) *dns.CNAME {
	return &dns.CNAME{
		Hdr: dns.RR_Header{
			Name:   dns.Fqdn(name),
			Rrtype: dns.TypeCNAME,
			Class:  dns.ClassINET,
		},
		Target: dns.Fqdn(value),
	}
}
