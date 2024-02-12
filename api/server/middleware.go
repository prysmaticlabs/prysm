package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/cors"
)

var (
	httpRequestLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_latency_milliseconds",
			Help:    "Latency of HTTP requests",
			Buckets: []float64{1, 10, 25, 100, 250, 1000, 2500, 10000},
		},
		[]string{"endpoint"},
	)
	httpRequestCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_request_count",
			Help: "Number of all HTTP requests",
		},
		[]string{"endpoint"},
	)
	failedHTTPRequestCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "failed_http_request_count",
			Help: "Number of failed HTTP requests",
		},
		[]string{"endpoint", "code"},
	)
)

// NormalizeQueryValuesHandler normalizes an input query of "key=value1,value2,value3" to "key=value1&key=value2&key=value3"
func NormalizeQueryValuesHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		NormalizeQueryValues(query)
		r.URL.RawQuery = query.Encode()

		next.ServeHTTP(w, r)
	})
}

// CorsHandler sets the cors settings on api endpoints
func CorsHandler(allowOrigins []string) mux.MiddlewareFunc {
	c := cors.New(cors.Options{
		AllowedOrigins:   allowOrigins,
		AllowedMethods:   []string{http.MethodPost, http.MethodGet, http.MethodDelete, http.MethodOptions},
		AllowCredentials: true,
		MaxAge:           600,
		AllowedHeaders:   []string{"*"},
	})

	return c.Handler
}

func PrometheusHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		next.ServeHTTP(w, r)
		route := mux.CurrentRoute(r)
		path, err := route.GetPathTemplate()
		if err != nil {
			httpRequestCount.WithLabelValues(path).Inc()
			if success(r.Response) {
				httpRequestLatency.WithLabelValues(path).Observe(float64(time.Since(now).Milliseconds()))
			} else {
				failedHTTPRequestCount.WithLabelValues(path).Inc()
			}
		}
	})
}

func success(resp *http.Response) bool {
	if resp == nil {
		return false
	}
	return strings.HasPrefix(resp.Status, "2")
}
