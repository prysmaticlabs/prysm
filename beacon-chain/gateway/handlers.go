package main

import (
	"fmt"
	"log"
	"net/http"
	"path"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

// swaggerServer returns swagger specification files located under "/swagger/"
func swaggerServer(dir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, ".swagger.json") {
			log.Printf("Not Found: %s\n", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		log.Printf("Serving %s\n", r.URL.Path)
		p := strings.TrimPrefix(r.URL.Path, "/swagger/")
		p = path.Join(dir, p)
		http.ServeFile(w, r, p)
	}
}

// healthzServer returns a simple health handler which returns ok.
func healthzServer(conn *grpc.ClientConn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		if s := conn.GetState(); s != connectivity.Ready {
			http.Error(w, fmt.Sprintf("grpc server is %s", s), http.StatusBadGateway)
			return
		}
		if _, err := fmt.Fprintln(w, "ok"); err != nil {
			panic(err) // TODO
		}
	}
}
