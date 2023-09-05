package dnstestserver

import (
	"context"
	"net/http"
	"strings"

	"github.com/miekg/dns"

	"github.com/renderinc/dns-test-server/pkg/logger"
	"github.com/renderinc/dns-test-server/pkg/rr"
)

type HTTPServer struct {
	Address string
	s       *RRStore
	l       logger.Logger
	srv     *http.Server
}

func NewHTTPServer(s *RRStore, address string) *HTTPServer {
	return &HTTPServer{
		Address: address,
		s:       s,
		l:       logger.NewStdLogger(),
		srv:     &http.Server{Addr: address},
	}
}

func (s *HTTPServer) ListenAndServe() error {
	s.srv.Handler = s
	return s.srv.ListenAndServe()
}

func (s *HTTPServer) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

func (s *HTTPServer) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	method := req.Method
	if method == "" {
		method = http.MethodGet
	}
	s.l.Infof("Serving HTTP request %s %s", method, req.RequestURI)

	wr.Header().Set("Content-Type", "text/plain")

	path := strings.TrimLeft(req.URL.Path, "/")
	if path == "" {
		if method == http.MethodHead {
			wr.WriteHeader(http.StatusOK)
			return
		}
		if method == http.MethodGet {
			wr.Write([]byte("OK"))
			return
		}
	}

	// GET /v1/a/www.google.com
	// PUT /v1/a/www.google.com?v=127.0.0.1
	// DELETE /v1/a/www.google.com
	parts := strings.SplitN(path, "/", 3)
	if len(parts) != 3 || parts[0] != "v1" {
		wr.WriteHeader(http.StatusNotFound)
		return
	}
	var recordType uint16
	name := parts[2]

	switch parts[1] {
	case "a":
		recordType = dns.TypeA
	case "cname":
		recordType = dns.TypeCNAME
	default:
		wr.WriteHeader(http.StatusBadRequest)
		return
	}

	switch method {
	case http.MethodGet:
		rs := s.s.Find(name, recordType)
		if len(rs) == 0 {
			wr.WriteHeader(http.StatusNotFound)
			return
		}
		for _, r := range rs {
			_, _ = wr.Write([]byte(r.String()))
			_, _ = wr.Write([]byte("\n"))
		}
	case http.MethodPut:
		v := req.URL.Query().Get("v")
		if v == "" {
			wr.WriteHeader(http.StatusBadRequest)
			return
		}
		r, err := rr.New(recordType, name, v)
		if rr.IsInvalidArg(err) {
			wr.WriteHeader(http.StatusBadRequest)
			return
		}
		if err != nil {
			s.l.Infof("unhandled error: %v", err)
			wr.WriteHeader(http.StatusInternalServerError)
			return
		}
		if existed := s.s.Add(r); existed {
			wr.WriteHeader(http.StatusNoContent)
		} else {
			wr.WriteHeader(http.StatusCreated)
		}
	case http.MethodDelete:
		s.s.Remove(name, recordType)
		wr.WriteHeader(http.StatusNoContent)
	default:
		wr.WriteHeader(http.StatusMethodNotAllowed)
	}
}
