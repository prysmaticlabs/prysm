package rpc

import (
	"net/http"
	"testing"

	"github.com/gorilla/mux"
	pb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

var _ pb.AuthServer = (*Server)(nil)

func TestServer_InitializeRoutes(t *testing.T) {
	s := Server{
		router: mux.NewRouter(),
	}
	err := s.InitializeRoutes()
	require.NoError(t, err)

	wantRouteList := map[string][]string{
		"/eth/v1/remotekeys":                        {http.MethodGet, http.MethodPost, http.MethodDelete},
		"/eth/v1/validator/{pubkey}/gas_limit":      {http.MethodGet, http.MethodPost, http.MethodDelete},
		"/eth/v1/validator/{pubkey}/voluntary_exit": {http.MethodPost},
	}
	gotRouteList := make(map[string][]string)
	err = s.router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		tpl, err1 := route.GetPathTemplate()
		require.NoError(t, err1)
		met, err2 := route.GetMethods()
		require.NoError(t, err2)
		methods, ok := gotRouteList[tpl]
		if !ok {
			gotRouteList[tpl] = []string{met[0]}
		} else {
			gotRouteList[tpl] = append(methods, met[0])
		}
		return nil
	})
	require.NoError(t, err)
	require.DeepEqual(t, wantRouteList, gotRouteList)
}
