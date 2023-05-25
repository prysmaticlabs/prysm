package apimiddleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/prysmaticlabs/prysm/v4/api/gateway/apimiddleware"
)

// "/eth/v1/validator/{pubkey}/voluntary_exit" POST expects epoch as a query param.
// This hook adds the query param to the body so that it is a valid POST request as
// grpc-gateway does not handle query params in POST requests.
func setVoluntaryExitEpoch(
	endpoint *apimiddleware.Endpoint,
	_ http.ResponseWriter,
	req *http.Request,
) (apimiddleware.RunDefault, apimiddleware.ErrorJson) {
	if _, ok := endpoint.PostRequest.(*SetVoluntaryExitRequestJson); ok {
		var epoch = req.URL.Query().Get("epoch")
		// To handle the request without the query param
		if epoch == "" {
			epoch = "0"
		}
		_, err := strconv.ParseUint(epoch, 10, 64)
		if err != nil {
			return false, apimiddleware.InternalServerErrorWithMessage(err, "invalid epoch")
		}
		j := &SetVoluntaryExitRequestJson{Epoch: epoch}
		b, err := json.Marshal(j)
		if err != nil {
			return false, apimiddleware.InternalServerErrorWithMessage(err, "could not marshal epoch")
		}
		req.Body = io.NopCloser(bytes.NewReader(b))
	}
	return true, nil
}
