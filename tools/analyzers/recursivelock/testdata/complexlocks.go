// These nested rlock patterns are too complex for the analyzer to catch right now
package testdata

func (p *ProtectResource) FuncLitInStructLit() {
	p.RLock()
	type funcLitContainer struct {
		funcLit func()
	}
	var fl *funcLitContainer = &funcLitContainer{
		funcLit: func() {
			p.RLock()
		},
	}
	fl.funcLit() // this is a nested RLock but won't be caught
	p.RUnlock()
}

func (e *ExposedMutex) FuncReturnsMutex() {
	e.GetLock().RLock()
	e.lock.RLock() // this is an obvious nested lock, but won't be caught since the first RLock was called through a getter function
	e.lock.RUnlock()
	e.GetLock().RUnlock()
}
