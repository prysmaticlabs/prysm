package proxy

import "net/http"

type InterceptorFunc func(reqBytes []byte, w http.ResponseWriter, r *http.Request) bool

func (pn *Proxy) AddInterceptor(icptr InterceptorFunc) {
	pn.interceptor = icptr
}

func (pn *Proxy) SyncingInterceptor() InterceptorFunc {
	return func(reqBytes []byte, w http.ResponseWriter, r *http.Request) bool {
		if !pn.checkIfValid(reqBytes) {
			return false
		}
		pn.returnSyncingResponse(reqBytes, w, r)
		return true
	}
}
