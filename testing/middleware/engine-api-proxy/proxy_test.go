package proxy

import "testing"

func TestProxy(t *testing.T) {
	t.Run("fails to proxy if destination is down", func(t *testing.T) {

	})
	t.Run("properly proxies request/response", func(t *testing.T) {

	})
}

func TestProxy_CustomInterceptors(t *testing.T) {
	t.Run("only intercepts engine API methods", func(t *testing.T) {

	})
	t.Run("only intercepts if trigger function returns true", func(t *testing.T) {

	})
	t.Run("triggers interceptor response correctly", func(t *testing.T) {

	})
}

func Test_isEngineAPICall(t *testing.T) {
	t.Run("naked array returns false", func(t *testing.T) {

	})
	t.Run("non-engine call returns false", func(t *testing.T) {

	})
	t.Run("engine call returns true", func(t *testing.T) {

	})
}
