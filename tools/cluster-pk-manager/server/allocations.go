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
		a := s.db.Allocations()
		for podName, pubkeys := range a {
			res += podName + "=["
			for _, pk := range pubkeys {
				res += fmt.Sprintf("%#x,", pk)
			}
			res += "]\n"
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, res)
	})
}
