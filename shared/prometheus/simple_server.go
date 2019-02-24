package prometheus

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// RunSimpleServerOrDie is a blocking call to serve /metrics at the given
// address.
func RunSimpleServerOrDie(addr string) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	svr := &http.Server{Addr: addr, Handler: mux}

	if err := svr.ListenAndServe(); err != nil {
		panic(err)
	}
}
