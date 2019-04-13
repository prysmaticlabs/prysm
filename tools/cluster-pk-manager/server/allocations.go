package main

import (
	"fmt"
	"io"
	"net/http"
)

func (s *server) serveAllocationsHTTPPage() {
	mux := http.NewServeMux()

	mux.HandleFunc("/allocations", func(w http.ResponseWriter, _ *http.Request) {
		res := ""
		a, err := s.db.Allocations()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, err.Error())
			return
		}
		for podName, pubkeys := range a {
			for _, pk := range pubkeys {
				res += fmt.Sprintf("%s=%#x\n", podName, pk)
			}
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, res)
	})

	srv := &http.Server{Addr: ":8080", Handler: mux}
	go srv.ListenAndServe()
	log.Info("Serving allocations page at :8080/allocations")
}
