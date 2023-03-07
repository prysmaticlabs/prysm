package web

import (
	"net/http"
)

const prefix = "external/prysm_web_ui/prysm-web-ui"

// Handler serves web requests from the bundled site data.
// DEPRECATED: Prysm Web UI and associated endpoints will be fully removed in a future hard fork.
var Handler = func(res http.ResponseWriter, req *http.Request) {
}
