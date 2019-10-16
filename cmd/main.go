package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sys/unix"

	"github.com/renderinc/dns-test-server/pkg/dnstestserver"
	"github.com/renderinc/dns-test-server/pkg/logger"
)

const (
	httpPortEnvVarKey = "DNS_HTTP_PORT"
	dnsPortEnvVarKey  = "DNS_PORT"
)

var log logger.Logger = logger.NewStdLogger()

func main() {
	if err := mainE(); err != nil {
		log.Info(err)
		os.Exit(1)
	}
}

func mainE() error {
	httpAddr := ":8080"
	if p := os.Getenv(httpPortEnvVarKey); p != "" {
		if _, err := strconv.Atoi(p); err != nil {
			return fmt.Errorf("invalid HTTP port: %s", p)
		}
		httpAddr = ":" + p
	}

	dnsAddr := ":8053"
	if p := os.Getenv(dnsPortEnvVarKey); p != "" {
		if _, err := strconv.Atoi(p); err != nil {
			return fmt.Errorf("invalid DNS port: %s", p)
		}
		dnsAddr = ":" + p
	}

	ims := &dnstestserver.RRStore{}
	httpSrv := dnstestserver.NewHTTPServer(ims, httpAddr)
	dnsSrv, err := dnstestserver.NewDNSServer(ims, dnsAddr)
	if err != nil {
		return err
	}

	errCh := make(chan error, 1)

	startServers(errCh, dnsSrv, httpSrv)
	defer shutdown(30*time.Second, dnsSrv, httpSrv)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, unix.SIGTERM, unix.SIGINT)

	select {
	case err = <-errCh:
	case sig := <-sigCh:
		log.Infof("Received signal %s", sig)
	}

	return err
}

func startServers(errCh chan<- error, dnsSrv *dnstestserver.DNSServer, httpSrv *dnstestserver.HTTPServer) {
	go func() {
		log.Infof("Listening to HTTP at %s", httpSrv.Address)
		err := httpSrv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Infof("HTTP server exited with %v", err)
			errCh <- err
		}
		errCh <- nil
	}()

	go func() {
		log.Infof("Listening to DNS at %s", dnsSrv.Address)
		err := dnsSrv.ListenAndServe()
		if err != nil {
			log.Infof("DNS server exited with %v", err)
		}
		errCh <- err
	}()
}

func shutdown(gracePeriod time.Duration, dnsSrv *dnstestserver.DNSServer, httpSrv *dnstestserver.HTTPServer) {
	ctx, cancel := context.WithTimeout(context.Background(), gracePeriod)
	defer cancel()

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		_ = dnsSrv.Shutdown(ctx)
	}()
	go func() {
		defer wg.Done()
		_ = httpSrv.Shutdown(ctx)
	}()

	wg.Wait()
}
