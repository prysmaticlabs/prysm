// These are all non recursive rlocks. Testing to make sure there are no false positives
package testdata

func (resource *ProtectResource) NestedRLockWithDefer() string {
	resource.RLock()
	defer resource.RUnlock()
	return resource.GetResource() // want `found recursive read lock call`
}

func (resource *NestedProtectResource) NonNestedRLockDifferentRLocks() {
	resource.RLock()
	resource.GetNestedPResource() // get nested resource uses RLock, but at a deeper level in the struct
	resource.RUnlock()
}
