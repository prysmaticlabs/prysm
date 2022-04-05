// recursive read lock calls with methods
package testdata

func (p *ProtectResource) NestedMethod() {
	p.RLock()
	p.GetResource() // want `found recursive read lock call`
	p.RUnlock()
}

func (p *ProtectResource) NestedMethod2() {
	p.RLock()
	p.GetResourceNested() // want `found recursive read lock call`
	p.RUnlock()
}

func (p *NestedProtectResource) MultiLevelStruct() {
	p.nestedPR.RLock()
	p.nestedPR.GetResource() // want `found recursive read lock call`
	p.nestedPR.RUnlock()
}

func (p *NestedProtectResource) MultiLevelStruct2() {
	p.nestedPR.RLock()
	p.GetNestedPResource() // want `found recursive read lock call`
	p.nestedPR.RUnlock()
}
