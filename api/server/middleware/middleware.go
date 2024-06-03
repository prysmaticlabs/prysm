package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
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

// ContentTypeHandler checks request for the appropriate media types otherwise returning a http.StatusUnsupportedMediaType error
func ContentTypeHandler(acceptedMediaTypes []string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			contentType := r.Header.Get("Content-Type")
			if contentType == "" {
				http.Error(w, "Content-Type header is missing", http.StatusUnsupportedMediaType)
				return
			}

			accepted := false
			for _, acceptedType := range acceptedMediaTypes {
				if strings.TrimSpace(contentType) == strings.TrimSpace(acceptedType) {
					accepted = true
					break
				}
			}

			if !accepted {
				http.Error(w, fmt.Sprintf("Unsupported media type: %s", contentType), http.StatusUnsupportedMediaType)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
