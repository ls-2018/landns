package landns

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"

	"github.com/macrat/landns/lib-landns/logger/httplog"
	"github.com/miekg/dns"
)

// Server is the Landns server instance.
type Server struct {
	Name            string
	Metrics         *Metrics
	DynamicResolver DynamicResolver
	Resolvers       Resolver // Resolvers for this server. Must include DynamicResolver.
	DebugMode       bool
}

// HTTPHandler is getter of http.Handler.
func (s *Server) HTTPHandler() (http.Handler, error) {
	mux := http.NewServeMux()

	serverName := s.Name
	if serverName == "" {
		serverName = "Landns"
	}

	if !s.DebugMode {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "<h1>%s</h1><a href=\"/metrics\">metrics</a> <a href=\"/api/v1\">records</a>\n", serverName)
		})
	} else {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "<h1>%s</h1><a href=\"/metrics\">metrics</a> <a href=\"/debug/pprof/\">pprof</a> <a href=\"/api/v1\">records</a>\n", serverName)
		})

		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
		mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
		mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
		mux.Handle("/debug/pprof/trace", pprof.Handler("trace"))
	}

	metrics, err := s.Metrics.HTTPHandler()
	if err != nil {
		return nil, Error{TypeExternalError, err, "failed to get Prometheus HTTP handler"}
	}

	mux.Handle("/metrics", metrics)
	if s.DynamicResolver != nil {
		mux.Handle("/api/", http.StripPrefix("/api", DynamicAPI{s.DynamicResolver}.Handler()))
	}

	return httplog.HTTPLogger{Handler: mux}, nil
}

// DNSHandler is getter of dns.Handler of package github.com/miekg/dns
func (s *Server) DNSHandler() dns.Handler {
	return NewHandler(s.Resolvers, s.Metrics)
}

// ListenAndServe is starter of server.
func (s *Server) ListenAndServe(ctx context.Context, apiAddress *net.TCPAddr, dnsAddress *net.UDPAddr, dnsProto string) error {
	httpHandler, err := s.HTTPHandler()
	if err != nil {
		return err
	}
	httpServer := http.Server{
		Addr:    apiAddress.String(),
		Handler: httpHandler,
	}

	dnsServer := dns.Server{
		Addr:      dnsAddress.String(),
		Net:       dnsProto,
		ReusePort: true,
		Handler:   s.DNSHandler(),
	}

	httpch := make(chan error)
	dnsch := make(chan error)
	defer close(httpch)
	defer close(dnsch)

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			httpch <- err
		}
	}()
	go func() {
		if err := dnsServer.ListenAndServe(); err != nil {
			dnsch <- err
		}
	}()

	select {
	case err = <-httpch:
		dnsServer.ShutdownContext(ctx)
		return Error{TypeInternalError, err, "fatal error on HTTP server"}
	case err = <-dnsch:
		httpServer.Shutdown(ctx)
		return Error{TypeInternalError, err, "fatal error on DNS server"}
	case <-ctx.Done():
		dnsServer.ShutdownContext(ctx)
		httpServer.Shutdown(ctx)
		return nil
	}
}
