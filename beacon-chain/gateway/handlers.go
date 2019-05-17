package gateway

import (
	"net/http"
	"path"
	"strings"
)

// Swagger directory for the runtime files provided by bazel data.
const swaggerDir = "proto/beacon/rpc/v1/"

// SwaggerServer returns swagger specification files located under "/swagger/"
func SwaggerServer() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, ".swagger.json") {
			log.Debugf("Not Found: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		log.Debugf("Serving %s\n", r.URL.Path)
		p := strings.TrimPrefix(r.URL.Path, "/swagger/")
		p = path.Join(swaggerDir, p)
		http.ServeFile(w, r, p)
	}
}
