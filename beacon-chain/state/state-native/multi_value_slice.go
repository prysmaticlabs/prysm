package state_native

import (
	"fmt"
	"sync"

	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
)

type MultiValue[V any, O any] struct {
	shared     V
	individual []*Value[V, O]
}

type Value[V any, O any] struct {
	val  V
	objs []O
}

type MultiValueSlice[V comparable, O comparable] struct {
	items []*MultiValue[V, O]
	lock  sync.RWMutex
}

func (r *MultiValueSlice[V, O]) Len() int {
	return len(r.items)
}

func (r *MultiValueSlice[V, O]) Copy(src O, dst O) {
	r.lock.Lock()
	defer r.lock.Unlock()

	for _, item := range r.items {
		if item.individual != nil {
		outerLoop:
			for _, mv := range item.individual {
				for _, o := range mv.objs {
					if o == src {
						mv.objs = append(mv.objs, dst)
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

	v := make([]V, len(r.items))
	for i, item := range r.items {
		if item.individual == nil {
			v[i] = r.items[i].shared
		} else {
			found := false
		outerLoop:
			for _, mv := range item.individual {
				for _, o := range mv.objs {
					if o == obj {
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

	item := r.items[i]
	if item.individual == nil {
		return item.shared, nil
	}
	for _, mv := range item.individual {
		for _, o := range mv.objs {
			if o == obj {
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

outerLoop:
	for mvi, mv := range item.individual {
		for oi, o := range mv.objs {
			if o == obj {
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
			mv.objs = append(mv.objs, obj)
			newValue = false
			break
		}
	}
	if newValue {
		item.individual = append(item.individual, &Value[V, O]{val: val, objs: []O{obj}})
	}

	return nil
}

/*func (r *MultiValueSlice[V, O]) Leave(obj O) {
	//log.Warnf("participants: %d", len(r.participants))
	for pi, p := range r.participants {
		if p == obj {
			for _, item := range r.items {
			outerLoop:
				for mvi, mv := range item.individual {
					for oi, o := range mv.objs {
						if o == obj {
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
			r.participants = append(r.participants[:pi], r.participants[pi+1:]...)
			break
		}
	}
}*/

type MultiValueRandaoMixes = MultiValueSlice[[32]byte, *BeaconState]

func NewMultiValueRandaoMixes(mixes [][]byte) *MultiValueRandaoMixes {
	items := make([]*MultiValue[[32]byte, *BeaconState], fieldparams.RandaoMixesLength)
	for i, b := range mixes {
		items[i] = &MultiValue[[32]byte, *BeaconState]{shared: *(*[32]byte)(b), individual: nil}
	}
	return &MultiValueRandaoMixes{
		items: items,
	}
}
