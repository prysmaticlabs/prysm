package proxy

import "net/http"

type InterceptorFunc func(reqBytes []byte, w http.ResponseWriter, r *http.Request) bool

func (p *Proxy) AddInterceptor(icptr InterceptorFunc) {
	p.interceptor = icptr
}

func (p *Proxy) SyncingInterceptor() InterceptorFunc {
	return func(reqBytes []byte, w http.ResponseWriter, r *http.Request) bool {
		if !checkIfValid(reqBytes) {
			return false
		}
		p.returnSyncingResponse(reqBytes, w, r)
		return true
	}
}
