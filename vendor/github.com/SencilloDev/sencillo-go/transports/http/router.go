// Copyright 2025 Sencillo
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/SencilloDev/sencillo-go/metrics"
	cwmiddleware "github.com/SencilloDev/sencillo-go/transports/http/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sagikazarmark/slog-shim"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/sdk/trace"
)

var ErrInternalError = fmt.Errorf("internal server error")

// ServerOption is a functional option to modify the server
type ServerOption func(*Server)

// handlerWithError is a normal HTTP handler but returns an error
type handlerWithError func(http.ResponseWriter, *http.Request) error

type MiddlewareWithLogger func(*Server, http.Handler) http.Handler

// errHandler contains a handler that returns an error and a logger
type ErrHandler struct {
	Handler handlerWithError
	Logger  *slog.Logger
}

// Server holds the http.Server, a logger, and the router to attach to the http.Server
type Server struct {
	apiServer      *http.Server
	Logger         *slog.Logger
	Router         *http.ServeMux
	Exporter       *metrics.Exporter
	traceShutdown  func(context.Context) error
	TracerProvider *trace.TracerProvider
}

// Route contains the information needed for an HTTP handler
type Route struct {
	Method  string
	Path    string
	Handler http.Handler
}

func JsonHandler(h handlerWithError) handlerWithError {
	return func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Add("Content-Type", "application/json")
		return h(w, r)
	}
}

func healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// NewHTTPServer initializes and returns a new Server
func NewHTTPServer(opts ...ServerOption) *Server {
	r := http.NewServeMux()

	s := &Server{
		Logger:   slog.New(slog.NewTextHandler(os.Stdout, nil)),
		Router:   r,
		Exporter: metrics.NewExporter(),
		apiServer: &http.Server{
			Addr:         ":8080",
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  10 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(s)
	}

	s.getHealth()
	s.apiServer.Handler = r

	s.Router.Handle("GET /metrics", promhttp.Handler())

	return s
}

func HandleWithContext[T any](h func(http.ResponseWriter, *http.Request, T), ctx T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h(w, r, ctx)
	}
}

func (s *Server) getHealth() {
	if s.TracerProvider != nil {
		s.Router.Handle("GET /healthz", otelhttp.NewHandler(http.HandlerFunc(healthz), "healthz:GET"))
		return
	}
	s.Router.Handle("GET /healthz", http.HandlerFunc(healthz))
}

// SetServerPort sets the server listening port
func SetServerPort(p int) ServerOption {
	return func(s *Server) {
		s.apiServer.Addr = fmt.Sprintf(":%d", p)
	}
}

// SetReadTimeout sets the http.Server read timeout
func SetReadTimeout(t int) ServerOption {
	return func(s *Server) {
		s.apiServer.ReadTimeout = time.Duration(t) * time.Second
	}
}

// SetWriteTimeout sets the http.Server write timeout
func SetWriteTimeout(t int) ServerOption {
	return func(s *Server) {
		s.apiServer.WriteTimeout = time.Duration(t) * time.Second
	}
}

// SetIdleTimeout sets the http.Server idle timeout
func SetIdleTimeout(t int) ServerOption {
	return func(s *Server) {
		s.apiServer.IdleTimeout = time.Duration(t) * time.Second
	}
}

func SetTracerProvider(t *trace.TracerProvider) ServerOption {
	return func(s *Server) {
		s.TracerProvider = t
	}
}

// ServeHTTP satisfies the http.Handler interface to allow for handling of errors from handlers in one place
func (e *ErrHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := e.Handler(w, r)
	if err == nil {
		return
	}

	var ce *ClientError
	if errors.As(err, &ce) {
		w.WriteHeader(ce.Status)
		w.Write([]byte(ce.Body()))
		return
	}

	e.Logger.Error(fmt.Sprintf("status=%d, err=%v", http.StatusInternalServerError, err))
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(ErrInternalError.Error()))
}

// RegisterSubRouter creates a subrouter based on a path and a slice of routes. Any middlewares passed in will be mounted to the sub router
func (s *Server) RegisterSubRouter(prefix string, routes []Route, middleware ...func(http.Handler) http.Handler) *Server {
	// HTTP Muxer requires the trailing slash for the prefix but hen we remove the slash in the strip prefix
	var prefixWithSlash string
	if strings.HasSuffix(prefix, "/") {
		prefixWithSlash = prefix
	} else {
		prefixWithSlash = fmt.Sprintf("%s/", prefix)
	}
	stripped := strings.TrimSuffix(prefix, "/")

	subRouter := http.NewServeMux()
	// we need to register each vector with a unique name, for now its a combination of the prefix and route path
	replacer := strings.NewReplacer("{", "", "}", "", "/", "_", "[", "_", "]", "_", "-", "_")
	name := fmt.Sprintf("%s%s", replacer.Replace(prefix), replacer.Replace(routes[0].Path))
	counter := metrics.NewCounterVec(fmt.Sprintf("http_requests%s", name), "HTTP requests by status, path, and method", []string{"code", "method", "path"})
	hist := metrics.NewHistogramVec(fmt.Sprintf("http_request_latency%s", name), "HTTP latency by status, path, and method", []string{"code", "method", "path"})

	reqWrapped := cwmiddleware.RequestID(subRouter)

	for _, m := range middleware {
		reqWrapped = m(reqWrapped)
	}

	// wrap subrouter to catch all middleware and total metrics for the subrouter
	for _, v := range routes {
		if s.traceShutdown != nil {
			m := fmt.Sprintf("%v:%v", v.Path, v.Method)
			subRouter.Handle(fmt.Sprintf("%s %s", v.Method, v.Path), otelhttp.NewHandler(v.Handler, m))
		} else {
			subRouter.Handle(fmt.Sprintf("%s %s", v.Method, v.Path), v.Handler)
		}
	}

	s.Exporter.Metrics = append(s.Exporter.Metrics, counter, hist)

	s.Router.Handle(prefixWithSlash, cwmiddleware.Logging(cwmiddleware.CodeStats(http.StripPrefix(stripped, reqWrapped), counter, hist)))

	return s
}

// Serve starts the http.Server
func (s *Server) Serve(errChan chan<- error) {
	prometheus.MustRegister(s.Exporter.Metrics...)

	s.Logger.Info(fmt.Sprintf("starting HTTP server on %s", s.apiServer.Addr))
	if err := s.apiServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		errChan <- err
	}
}

// AutoHandleErrors is a convenience that should be called after starting the server.
// It will automatically safely stop the server if a signal is received. This breaks
// the normal pattern of letting the caller handle fatal errors, which is why this is a convenience
// function that's able to be called separately.
func (s *Server) AutoHandleErrors(ctx context.Context, errChan <-chan error) {
	go func() {
		serverErr := <-errChan
		if serverErr != nil {
			s.Logger.Error(fmt.Sprintf("error starting server: %v", serverErr))
			s.ShutdownServer(ctx)
		}
	}()

	sigTerm := make(chan os.Signal, 1)
	signal.Notify(sigTerm, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigTerm
	s.Logger.Info(fmt.Sprintf("received signal: %s", sig))
	s.ShutdownServer(ctx)
}

func (s *Server) ShutdownServer(ctx context.Context) {
	s.Logger.Info("shutting down server")
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if s.TracerProvider != nil {
		if err := s.TracerProvider.Shutdown(ctx); err != nil {
			s.Logger.Error(fmt.Sprintf("error stopping tracing: %v\n", err))
		}
	}

	if err := s.apiServer.Shutdown(ctx); err != nil {
		s.Logger.Error(fmt.Sprintf("error shutting down server: %v\n", err))
	}

	s.Logger.Info("server stopped")
	os.Exit(1)
}
