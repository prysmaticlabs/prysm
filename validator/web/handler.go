package web

import (
	"net/http"
)

// Handler serves web requests from the bundled site data.
// DEPRECATED: Prysm Web UI and associated endpoints will be fully removed in a future hard fork.
var Handler = func(res http.ResponseWriter, req *http.Request) {
}
