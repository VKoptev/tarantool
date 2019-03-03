package debug

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/http/pprof"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Service for debugging
type Service struct {
	healthy *int32
}

// New debug interface
func New() *Service {
	var h int32
	return &Service{healthy: &h}
}

// SetHealthOK call this after your application is ready for incomming requests
func (s *Service) SetHealthOK() {
	atomic.StoreInt32(s.healthy, 1)
}

// SetHealthNotOK call this wheb your application is stopping or have problems
func (s *Service) SetHealthNotOK() {
	atomic.StoreInt32(s.healthy, 0)
}

// Serve debug information. It doesn't use default http server
func (s *Service) Serve(ctx context.Context, l string, logger *log.Logger) error {
	if logger == nil {
		return errors.New("you must use not nil logger")
	}
	r := http.NewServeMux()
	r.Handle("/metrics", promhttp.Handler())
	r.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	r.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	r.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	r.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	r.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
	r.Handle("/healthz", http.HandlerFunc(s.healthz))

	ms := http.Server{
		Addr:         l,
		Handler:      r,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
		ErrorLog:     logger,
	}
	e := make(chan error, 1)
	go func() {
		e <- ms.ListenAndServe()
	}()
	select {
	case <-ctx.Done():
		return ms.Shutdown(context.Background())
	case err := <-e:
		return err
	}
}

//nolint: unparam
func (s *Service) healthz(w http.ResponseWriter, r *http.Request) {
	if atomic.LoadInt32(s.healthy) == 0 {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("NotOK")) //nolint: errcheck, gas, gosec
		return
	}
	w.Write([]byte("OK")) //nolint: errcheck, gas, gosec
	w.WriteHeader(http.StatusOK)
}
