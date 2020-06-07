package dnstestserver

import (
	"context"
	"errors"

	"github.com/miekg/dns"

	"github.com/renderinc/dns-test-server/pkg/logger"
)

var notFound = errors.New("not found")

type DNSServer struct {
	Address      string
	s            *RRStore
	l            logger.Logger
	clientConfig *dns.ClientConfig
	client       *dns.Client
	started      chan bool
	srv          *dns.Server
}

func NewDNSServer(s *RRStore, address string) (*DNSServer, error) {
	cfg, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil {
		return nil, err
	}
	started := make(chan bool)
	return &DNSServer{
		Address:      address,
		s:            s,
		l:            logger.NewStdLogger(),
		clientConfig: cfg,
		client:       &dns.Client{},
		started:      started,
		srv: &dns.Server{
			Addr:              address,
			Net:               "udp",
			NotifyStartedFunc: func() { close(started) },
		},
	}, nil
}

// Started returns a channel that is closed when the DNSServer has started and is listening for connections.
//
// Started is provided for use in select statements
//
// Started can be used to ensure DNS requests are sent to a server only after it has been started.
func (s *DNSServer) Started() chan bool {
	return s.started
}

func (s *DNSServer) ListenAndServe() error {
	s.srv.Handler = s
	return s.srv.ListenAndServe()
}

func (s *DNSServer) Shutdown(ctx context.Context) error {
	return s.srv.ShutdownContext(ctx)
}

func (s *DNSServer) ServeDNS(wr dns.ResponseWriter, req *dns.Msg) {
	s.l.Infof("Serving DNS req %d", req.Id)
	var cnameRRs []dns.RR
	cnameForName := map[string]string{}

	for _, q := range req.Question {
		if q.Qclass != dns.ClassINET || q.Qtype == dns.TypeCNAME {
			// we'll only deal with Internet queries.
			// also, no recursive resolution of CNAME queries
			break
		}
		name, cname := q.Name, q.Name
		// resolve aliases if we're not dealing with a CNAME question
		for seen := map[string]bool{}; cname != "" && !seen[cname]; {
			seen[cname] = true // avoid CNAME loops
			r := s.s.Find(cname, dns.TypeCNAME)
			if r == nil {
				break
			}
			cnameRR := r.(*dns.CNAME)
			cnameRRs = append(cnameRRs, cnameRR)
			cname = cnameRR.Target
		}
		cnameForName[name] = cname
	}

	if err := s.serveDNSFromIMS(cnameRRs, cnameForName, wr, req); err == nil {
		return
	} else if err != notFound {
		s.l.Infof("error serving DNS from IMS: %v", err)
	}
	s.serveDNSWithClient(cnameRRs, cnameForName, wr, req)
}

func (s *DNSServer) serveDNSFromIMS(
	cnameRRs []dns.RR, cnameForName map[string]string, wr dns.ResponseWriter, req *dns.Msg,
) error {
	var records []dns.RR
	for _, q := range req.Question {
		if q.Qclass != dns.ClassINET {
			return notFound
		}
		name := q.Name
		if q.Qtype != dns.TypeCNAME {
			if cname := cnameForName[name]; cname != "" {
				name = cnameForName[name]
			}
		}
		record := s.s.Find(name, q.Qtype)
		if record == nil {
			return notFound
		}
		records = append(records, record)
	}
	s.writeMsg(wr, &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Id:                 req.Id,
			Response:           true,
			RecursionDesired:   req.RecursionDesired,
			RecursionAvailable: true,
			Rcode:              dns.RcodeSuccess,
		},
		Question: req.Question,
		Answer:   append(append(cnameRRs[:0:0], cnameRRs...), records...),
	})
	return nil
}

func (s *DNSServer) serveDNSWithClient(
	cnameRRs []dns.RR, cnameForName map[string]string, wr dns.ResponseWriter, req *dns.Msg,
) {
	// rewrite the questions to use the canonical names
	reqCopy := req.Copy()
	for i := range reqCopy.Question {
		q := &reqCopy.Question[i]
		if q.Qtype == dns.TypeCNAME {
			continue // no rewriting CNAME queries
		}
		if cname := cnameForName[q.Name]; cname != "" {
			q.Name = cname
		}
	}

	res := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Id:                 req.Id,
			Response:           true,
			RecursionDesired:   req.RecursionDesired,
			RecursionAvailable: true,
		},
		Question: req.Question,
	}

	for _, ns := range s.clientConfig.Servers {
		nsAddr := ns + ":" + s.clientConfig.Port
		msg, _, err := s.client.Exchange(reqCopy, nsAddr)
		if err != nil || msg.Rcode == dns.RcodeServerFailure {
			s.l.Infof("error exchanging with ns %s: %v", nsAddr, err)
			continue
		}
		res.Rcode = dns.RcodeSuccess
		res.Answer = append(append(cnameRRs[:0:0], cnameRRs...), msg.Answer...)
		s.writeMsg(wr, res)
		return
	}
	res.Rcode = dns.RcodeNameError
	s.writeMsg(wr, res)
}

func (s *DNSServer) writeMsg(wr dns.ResponseWriter, msg *dns.Msg) {
	if err := wr.WriteMsg(msg); err != nil {
		s.l.Infof("error writing message: %v", err)
	}
}
