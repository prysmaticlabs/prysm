package multi_value_slice

import (
	"fmt"
	"reflect"
	"sync"
)

type MultiValue[V any] struct {
	Shared     V
	Individual []*Value[V]
}

type Value[V any] struct {
	val  V
	objs []uintptr
}

type Slice[V comparable, O comparable] struct {
	Items []*MultiValue[V]
	lock  sync.RWMutex
}

func (r *Slice[V, O]) Len() int {
	return len(r.Items)
}

func (r *Slice[V, O]) Copy(src O, dst O) {
	r.lock.Lock()
	defer r.lock.Unlock()

	pSrc := reflect.ValueOf(src).Elem().Addr().Pointer()
	pDst := reflect.ValueOf(dst).Elem().Addr().Pointer()

	for _, item := range r.Items {
		if item.Individual != nil {
		outerLoop:
			for _, mv := range item.Individual {
				for _, o := range mv.objs {
					if o == pSrc {
						mv.objs = append(mv.objs, pDst)
						break outerLoop
					}
				}
			}
		}
	}
}

func (r *Slice[V, O]) Value(obj O) []V {
	r.lock.RLock()
	defer r.lock.RUnlock()

	p := reflect.ValueOf(obj).Elem().Addr().Pointer()

	v := make([]V, len(r.Items))
	for i, item := range r.Items {
		if item.Individual == nil {
			v[i] = r.Items[i].Shared
		} else {
			found := false
		outerLoop:
			for _, mv := range item.Individual {
				for _, o := range mv.objs {
					if o == p {
						v[i] = mv.val
						found = true
						break outerLoop
					}
				}
			}
			if !found {
				v[i] = r.Items[i].Shared
			}
		}
	}

	return v
}

func (r *Slice[V, O]) At(obj O, i uint64) (V, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	if i >= uint64(len(r.Items)) {
		var def V
		return def, fmt.Errorf("index %d is out of bounds", i)
	}

	p := reflect.ValueOf(obj).Elem().Addr().Pointer()

	item := r.Items[i]
	if item.Individual == nil {
		return item.Shared, nil
	}
	for _, mv := range item.Individual {
		for _, o := range mv.objs {
			if o == p {
				return mv.val, nil
			}
		}
	}
	return item.Shared, nil
}

func (r *Slice[V, O]) UpdateAt(obj O, i uint64, val V) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if i >= uint64(len(r.Items)) {
		return fmt.Errorf("index %d is out of bounds", i)
	}

	item := r.Items[i]

	p := reflect.ValueOf(obj).Elem().Addr().Pointer()

outerLoop:
	for mvi, mv := range item.Individual {
		for oi, o := range mv.objs {
			if o == p {
				if len(mv.objs) == 1 {
					item.Individual = append(item.Individual[:mvi], item.Individual[mvi+1:]...)
				} else {
					mv.objs = append(mv.objs[:oi], mv.objs[oi+1:]...)
				}
				break outerLoop
			}
		}
	}

	if val == item.Shared {
		return nil
	}

	newValue := true
	for _, mv := range item.Individual {
		if mv.val == val {
			mv.objs = append(mv.objs, p)
			newValue = false
			break
		}
	}
	if newValue {
		item.Individual = append(item.Individual, &Value[V]{val: val, objs: []uintptr{p}})
	}

	return nil
}

func (r *Slice[V, O]) Detach(obj O) {
	r.lock.Lock()
	defer r.lock.Unlock()

	p := reflect.ValueOf(obj).Elem().Addr().Pointer()

	for _, item := range r.Items {
	outerLoop:
		for mvi, mv := range item.Individual {
			for oi, o := range mv.objs {
				if o == p {
					if len(mv.objs) == 1 {
						item.Individual = append(item.Individual[:mvi], item.Individual[mvi+1:]...)
					} else {
						mv.objs = append(mv.objs[:oi], mv.objs[oi+1:]...)
					}
					break outerLoop
				}
			}
		}
	}
}
