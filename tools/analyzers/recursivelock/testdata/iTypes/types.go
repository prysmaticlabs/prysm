package iTypes

import "sync"

type AwesomeProtectedResource struct {
	sync.RWMutex
	resource string
}

func (a *AwesomeProtectedResource) GetResource() string {
	defer a.RUnlock()
	a.RLock()
	return a.resource
}

func (a *AwesomeProtectedResource) SetResource(r string) {
	defer a.Unlock()
	a.Lock()
	a.resource = r
}
