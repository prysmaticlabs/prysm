package node

import (
	"net/http"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/helpers"
)

func middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		helpers.NormalizeQueryValues(query)
		r.URL.RawQuery = query.Encode()

		next.ServeHTTP(w, r)
	})
}
