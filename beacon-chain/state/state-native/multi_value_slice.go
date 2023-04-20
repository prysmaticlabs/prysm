package state_native

import (
	"fmt"
	"reflect"
	"sync"

	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
)

type MultiValue[V any] struct {
	shared     V
	individual []*Value[V]
}

type Value[V any] struct {
	val  V
	objs []uintptr
}

type MultiValueSlice[V comparable, O comparable] struct {
	items []*MultiValue[V]
	lock  sync.RWMutex
}

func (r *MultiValueSlice[V, O]) Len() int {
	return len(r.items)
}

func (r *MultiValueSlice[V, O]) Copy(src O, dst O) {
	r.lock.Lock()
	defer r.lock.Unlock()

	pSrc := reflect.ValueOf(src).Elem().Addr().Pointer()
	pDst := reflect.ValueOf(dst).Elem().Addr().Pointer()

	for _, item := range r.items {
		if item.individual != nil {
		outerLoop:
			for _, mv := range item.individual {
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

func (r *MultiValueSlice[V, O]) Value(obj O) []V {
	r.lock.RLock()
	defer r.lock.RUnlock()

	p := reflect.ValueOf(obj).Elem().Addr().Pointer()

	v := make([]V, len(r.items))
	for i, item := range r.items {
		if item.individual == nil {
			v[i] = r.items[i].shared
		} else {
			found := false
		outerLoop:
			for _, mv := range item.individual {
				for _, o := range mv.objs {
					if o == p {
						v[i] = mv.val
						found = true
						break outerLoop
					}
				}
			}
			if !found {
				v[i] = r.items[i].shared
			}
		}
	}

	return v
}

func (r *MultiValueSlice[V, O]) At(obj O, i uint64) (V, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	if i >= uint64(len(r.items)) {
		var def V
		return def, fmt.Errorf("index %d is out of bounds", i)
	}

	p := reflect.ValueOf(obj).Elem().Addr().Pointer()

	item := r.items[i]
	if item.individual == nil {
		return item.shared, nil
	}
	for _, mv := range item.individual {
		for _, o := range mv.objs {
			if o == p {
				return mv.val, nil
			}
		}
	}
	return item.shared, nil
}

func (r *MultiValueSlice[V, O]) UpdateAt(obj O, i uint64, val V) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if i >= uint64(len(r.items)) {
		return fmt.Errorf("index %d is out of bounds", i)
	}

	item := r.items[i]

	p := reflect.ValueOf(obj).Elem().Addr().Pointer()

outerLoop:
	for mvi, mv := range item.individual {
		for oi, o := range mv.objs {
			if o == p {
				if len(mv.objs) == 1 {
					item.individual = append(item.individual[:mvi], item.individual[mvi+1:]...)
				} else {
					mv.objs = append(mv.objs[:oi], mv.objs[oi+1:]...)
				}
				break outerLoop
			}
		}
	}

	if val == item.shared {
		return nil
	}

	newValue := true
	for _, mv := range item.individual {
		if mv.val == val {
			mv.objs = append(mv.objs, p)
			newValue = false
			break
		}
	}
	if newValue {
		item.individual = append(item.individual, &Value[V]{val: val, objs: []uintptr{p}})
	}

	return nil
}

func (r *MultiValueSlice[V, O]) Detach(obj O) {
	p := reflect.ValueOf(obj).Elem().Addr().Pointer()

	for _, item := range r.items {
	outerLoop:
		for mvi, mv := range item.individual {
			for oi, o := range mv.objs {
				if o == p {
					if len(mv.objs) == 1 {
						item.individual = append(item.individual[:mvi], item.individual[mvi+1:]...)
					} else {
						mv.objs = append(mv.objs[:oi], mv.objs[oi+1:]...)
					}
					break outerLoop
				}
			}
		}
	}
}

type MultiValueRandaoMixes = MultiValueSlice[[32]byte, *BeaconState]

func NewMultiValueRandaoMixes(mixes [][]byte) *MultiValueRandaoMixes {
	items := make([]*MultiValue[[32]byte], fieldparams.RandaoMixesLength)
	for i, b := range mixes {
		items[i] = &MultiValue[[32]byte]{shared: *(*[32]byte)(b), individual: nil}
	}
	return &MultiValueRandaoMixes{
		items: items,
	}
}
