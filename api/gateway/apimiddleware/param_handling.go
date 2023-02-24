package apimiddleware

import (
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/mux"
	butil "github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	"github.com/wealdtech/go-bytesutil"
)

// HandleURLParameters processes URL parameters, allowing parameterized URLs to be safely and correctly proxied to grpc-gateway.
func HandleURLParameters(url string, req *http.Request, literals []string) ErrorJson {
	segments := strings.Split(url, "/")

segmentsLoop:
	for i, s := range segments {
		// We only care about segments which are parameterized.
		if isRequestParam(s) {
			// Don't do anything with parameters which should be forwarded literally to gRPC.
			for _, l := range literals {
				if s == "{"+l+"}" {
					continue segmentsLoop
				}
			}

			routeVar := mux.Vars(req)[s[1:len(s)-1]]
			bRouteVar := []byte(routeVar)
			if butil.IsHex(bRouteVar) {
				var err error
				bRouteVar, err = bytesutil.FromHexString(string(bRouteVar))
				if err != nil {
					return InternalServerErrorWithMessage(err, "could not process URL parameter")
				}
			}
			// Converting hex to base64 may result in a value which malforms the URL.
			// We use URLEncoding to safely escape such values.
			base64RouteVar := base64.URLEncoding.EncodeToString(bRouteVar)

			// Merge segments back into the full URL.
			splitPath := strings.Split(req.URL.Path, "/")
			splitPath[i] = base64RouteVar
			req.URL.Path = strings.Join(splitPath, "/")
		}
	}
	return nil
}

// HandleQueryParameters processes query parameters, allowing them to be safely and correctly proxied to grpc-gateway.
func HandleQueryParameters(req *http.Request, params []QueryParam) ErrorJson {
	queryParams := req.URL.Query()

	normalizeQueryValues(queryParams)

	for key, vals := range queryParams {
		for _, p := range params {
			if key == p.Name {
				if p.Hex {
					queryParams.Del(key)
					for _, v := range vals {
						b := []byte(v)
						if butil.IsHex(b) {
							var err error
							b, err = bytesutil.FromHexString(v)
							if err != nil {
								return InternalServerErrorWithMessage(err, "could not process query parameter")
							}
						}
						queryParams.Add(key, base64.URLEncoding.EncodeToString(b))
					}
				}
				if p.Enum {
					queryParams.Del(key)
					for _, v := range vals {
						// gRPC expects uppercase enum values.
						queryParams.Add(key, strings.ToUpper(v))
					}
				}
			}
		}
	}
	req.URL.RawQuery = queryParams.Encode()
	return nil
}

// isRequestParam verifies whether the passed string is a request parameter.
// Request parameters are enclosed in { and }.
func isRequestParam(s string) bool {
	return len(s) > 2 && s[0] == '{' && s[len(s)-1] == '}'
}

func normalizeQueryValues(queryParams url.Values) {
	// Replace comma-separated values with individual values.
	for key, vals := range queryParams {
		splitVals := make([]string, 0)
		for _, v := range vals {
			splitVals = append(splitVals, strings.Split(v, ",")...)
		}
		queryParams[key] = splitVals
	}
}
