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

package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/segmentio/ksuid"
)

func Logging(h http.Handler) http.Handler {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	fn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		defer func() {
			logger.Info("path", r.URL.String(), "host", r.Host, fmt.Sprintf("request duration: %dms", time.Since(start).Milliseconds()))
		}()

		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func RequestID(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Request-ID") == "" {
			id := ksuid.New()
			r.Header.Add("X-Request-ID", id.String())
			w.Header().Add("X-Request-ID", id.String())
		}

		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

// StatusRec wraps the http.ResponseWriter to capture the status code
type StatusRec struct {
	http.ResponseWriter
	Status int
}

// WriteHeader captures the status code
func (r *StatusRec) WriteHeader(status int) {
	r.Status = status
	r.ResponseWriter.WriteHeader(status)
}

// CodesStats is a middleware that captures the status code and method of the request for metrics collection with Prometheus
func CodeStats(h http.Handler, vec *prometheus.CounterVec, hist *prometheus.HistogramVec) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		rec := &StatusRec{
			ResponseWriter: w,
			Status:         200,
		}
		start := time.Now()
		h.ServeHTTP(rec, r)

		vec.WithLabelValues(fmt.Sprintf("%d", rec.Status), r.Method, r.URL.Path).Inc()
		hist.WithLabelValues(fmt.Sprintf("%d", rec.Status), r.Method, r.URL.Path).Observe(float64(time.Since(start).Seconds()))
	}

	return http.HandlerFunc(fn)
}
