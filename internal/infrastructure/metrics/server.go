package metrics

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server exposes the Prometheus scrape endpoint on its own listener,
// separate from the public API.
//
// It is deliberately NOT mounted on the main Fiber app. /metrics leaks
// internal topology (symbols, queue depths, route names, error rates)
// and has no authentication, so it must not share a port with anything
// reachable from the internet. Binding to the configured host - which
// defaults to a loopback/private address - keeps it reachable by a
// sidecar or node-local Prometheus without exposing it publicly.
type Server struct {
	enabled bool
	addr    string
	path    string
	http    *http.Server
	errCh   chan error
}

// Options configures the metrics listener. It mirrors config.MetricsConfig
// but is declared locally so that this infrastructure package does not
// depend on the application config package.
type Options struct {
	Enabled bool
	Host    string
	Port    int
	Path    string
}

const (
	defaultMetricsHost = "127.0.0.1"
	defaultMetricsPort = 9464
	defaultMetricsPath = "/metrics"
)

// NewServer builds a metrics server from the supplied options,
// applying defaults for any unset field.
func NewServer(opts Options) *Server {
	host := strings.TrimSpace(opts.Host)
	if host == "" {
		host = defaultMetricsHost
	}

	port := opts.Port
	if port <= 0 {
		port = defaultMetricsPort
	}

	path := strings.TrimSpace(opts.Path)
	if path == "" {
		path = defaultMetricsPath
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	mux := http.NewServeMux()
	mux.Handle(path, promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{
			// A broken collector should show up as a scrape error in
			// Prometheus, not as a silent gap in one series.
			ErrorHandling: promhttp.HTTPErrorOnError,
		},
	))

	// Cheap liveness probe for the metrics listener itself, so a
	// sidecar can tell "process is up but not scrapable" apart from
	// "process is down".
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))

	return &Server{
		enabled: opts.Enabled,
		addr:    addr,
		path:    path,
		errCh:   make(chan error, 1),
		http: &http.Server{
			Addr:              addr,
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       10 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
	}
}

// Enabled reports whether metrics collection is switched on.
func (s *Server) Enabled() bool {
	return s != nil && s.enabled
}

// Address returns the fully qualified scrape URL, for logging.
func (s *Server) Address() string {
	if s == nil {
		return ""
	}
	return "http://" + s.addr + s.path
}

// Start binds the listener and serves in the background.
//
// The bind happens synchronously so that a port clash fails startup
// loudly instead of disappearing into a goroutine. If metrics are
// disabled, Start is a no-op and returns nil.
func (s *Server) Start() error {
	if !s.Enabled() {
		return nil
	}

	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("bind metrics listener on %s: %w", s.addr, err)
	}

	go func() {
		if err := s.http.Serve(listener); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			s.errCh <- err
		}
		close(s.errCh)
	}()

	return nil
}

// Err exposes an asynchronous serve failure, if any.
func (s *Server) Err() <-chan error {
	return s.errCh
}

// Shutdown gracefully stops the metrics listener.
func (s *Server) Shutdown(ctx context.Context) error {
	if s == nil || s.http == nil || !s.enabled {
		return nil
	}
	return s.http.Shutdown(ctx)
}