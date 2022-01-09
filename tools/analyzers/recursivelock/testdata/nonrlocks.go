// These are all non recursive rlocks. Testing to make sure there are no false positives
package testdata

func (resource *ProtectResource) NonNestedRLockWithDefer() string {
	resource.RLock()
	defer resource.RUnlock()
	return resource.GetResource() // this is not a nested rlock because runlock is deferred
}

func (resource *NestedProtectResource) NonNestedRLockDifferentRLocks() {
	resource.RLock()
	resource.GetNestedPResource() // get nested resource uses RLock, but at a deeper level in the struct
	resource.RUnlock()
}
