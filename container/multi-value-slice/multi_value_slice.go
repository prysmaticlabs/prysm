package multi_value_slice

import (
	"fmt"
	"sync"

	"github.com/prysmaticlabs/prysm/v4/container/multi-value-slice/interfaces"
)

type MultiValueSlice[O interfaces.Identifiable] interface {
	Len(obj O) int
}

type Value[V any] struct {
	val  V
	objs []interfaces.Id
}

type MultiValue[V any] struct {
	Individual []*Value[V]
}

type Slice[V comparable, O interfaces.Identifiable] struct {
	sharedItems     []V
	individualItems map[uint64]*MultiValue[V]
	appendedItems   []*MultiValue[V]
	cachedLengths   map[interfaces.Id]int
	lock            sync.RWMutex
}

func (s *Slice[V, O]) Init(items []V) {
	s.sharedItems = items
	s.individualItems = map[interfaces.Id]*MultiValue[V]{}
	s.appendedItems = []*MultiValue[V]{}
	s.cachedLengths = map[interfaces.Id]int{}
}

func (s *Slice[V, O]) Len(obj O) int {
	s.lock.RLock()
	defer s.lock.RUnlock()

	l, ok := s.cachedLengths[obj.Id()]
	if !ok {
		return len(s.sharedItems)
	}
	return l
}

func (s *Slice[V, O]) Copy(src O, dst O) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, item := range s.individualItems {
	individualLoop:
		for _, mv := range item.Individual {
			for _, o := range mv.objs {
				if o == src.Id() {
					mv.objs = append(mv.objs, dst.Id())
					break individualLoop
				}
			}
		}
	}

appendedLoop:
	for _, item := range s.appendedItems {
		found := false
	individualLoop2:
		for _, mv := range item.Individual {
			for _, o := range mv.objs {
				if o == src.Id() {
					found = true
					mv.objs = append(mv.objs, dst.Id())
					break individualLoop2
				}
			}
		}
		if !found {
			break appendedLoop
		}
	}

	srcLen, ok := s.cachedLengths[src.Id()]
	if ok {
		s.cachedLengths[dst.Id()] = srcLen
	}
}

func (s *Slice[V, O]) Value(obj O) []V {
	s.lock.RLock()
	defer s.lock.RUnlock()

	v := make([]V, len(s.sharedItems))
	for i, item := range s.sharedItems {
		ind, ok := s.individualItems[uint64(i)]
		if !ok {
			v[i] = item
		} else {
			found := false
		individualLoop:
			for _, mv := range ind.Individual {
				for _, o := range mv.objs {
					if o == obj.Id() {
						v[i] = mv.val
						found = true
						break individualLoop
					}
				}
			}
			if !found {
				v[i] = item
			}
		}
	}

	for _, item := range s.appendedItems {
		found := false
	individualLoop2:
		for _, mv := range item.Individual {
			for _, o := range mv.objs {
				if o == obj.Id() {
					found = true
					v = append(v, mv.val)
					break individualLoop2
				}
			}
		}
		if !found {
			return v
		}
	}

	return v
}

func (s *Slice[V, O]) At(obj O, i uint64) (V, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if i >= uint64(len(s.sharedItems)+len(s.appendedItems)) {
		var def V
		return def, fmt.Errorf("index %d out of bounds", i)
	}

	isOriginal := i < uint64(len(s.sharedItems))
	if isOriginal {
		ind, ok := s.individualItems[i]
		if !ok {
			return s.sharedItems[i], nil
		}
		for _, mv := range ind.Individual {
			for _, o := range mv.objs {
				if o == obj.Id() {
					return mv.val, nil
				}
			}
		}
		return s.sharedItems[i], nil
	} else {
		item := s.appendedItems[i-uint64(len(s.sharedItems))]
		for _, mv := range item.Individual {
			for _, o := range mv.objs {
				if o == obj.Id() {
					return mv.val, nil
				}
			}
		}
		var def V
		return def, fmt.Errorf("index %d out of bounds", i)
	}
}

func (s *Slice[V, O]) UpdateAt(obj O, i uint64, val V) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if i >= uint64(len(s.sharedItems)+len(s.appendedItems)) {
		return fmt.Errorf("index %d out of bounds", i)
	}

	isOriginal := i < uint64(len(s.sharedItems))
	if isOriginal {
		ind, ok := s.individualItems[i]
		if ok {
		individualLoop:
			for mvi, mv := range ind.Individual {
				for oi, o := range mv.objs {
					if o == obj.Id() {
						if len(mv.objs) == 1 {
							// There is an improvement to be made here. If len(ind.Individual) == 1,
							// then after removing the item from the slice s.individualItems[i]
							// will be a useless map entry whose value is an empty slice.
							ind.Individual = append(ind.Individual[:mvi], ind.Individual[mvi+1:]...)
						} else {
							mv.objs = append(mv.objs[:oi], mv.objs[oi+1:]...)
						}
						break individualLoop
					}
				}
			}
		}

		if val == s.sharedItems[i] {
			return nil
		}

		if !ok {
			s.individualItems[i] = &MultiValue[V]{Individual: []*Value[V]{{val: val, objs: []uint64{obj.Id()}}}}
		} else {
			newValue := true
			for _, mv := range ind.Individual {
				if mv.val == val {
					mv.objs = append(mv.objs, obj.Id())
					newValue = false
					break
				}
			}
			if newValue {
				ind.Individual = append(ind.Individual, &Value[V]{val: val, objs: []uint64{obj.Id()}})
			}
		}
	} else {
		item := s.appendedItems[i-uint64(len(s.sharedItems))]
		found := false
	individualLoop2:
		for mvi, mv := range item.Individual {
			for oi, o := range mv.objs {
				if o == obj.Id() {
					found = true
					if len(mv.objs) == 1 {
						item.Individual = append(item.Individual[:mvi], item.Individual[mvi+1:]...)
					} else {
						mv.objs = append(mv.objs[:oi], mv.objs[oi+1:]...)
					}
					break individualLoop2
				}
			}
		}
		if !found {
			return fmt.Errorf("index %d out of bounds", i)
		}

		newValue := true
		for _, mv := range item.Individual {
			if mv.val == val {
				mv.objs = append(mv.objs, obj.Id())
				newValue = false
				break
			}
		}
		if newValue {
			item.Individual = append(item.Individual, &Value[V]{val: val, objs: []uint64{obj.Id()}})
		}
	}

	return nil
}

func (s *Slice[V, O]) Append(obj O, val V) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if len(s.appendedItems) == 0 {
		s.appendedItems = append(s.appendedItems, &MultiValue[V]{Individual: []*Value[V]{{val: val, objs: []uint64{obj.Id()}}}})
		s.cachedLengths[obj.Id()] = len(s.sharedItems) + 1
		return
	}

	for _, item := range s.appendedItems {
		found := false
	individualLoop:
		for _, mv := range item.Individual {
			for _, o := range mv.objs {
				if o == obj.Id() {
					found = true
					break individualLoop
				}
			}
		}
		if !found {
			newValue := true
			for _, mv := range item.Individual {
				if mv.val == val {
					mv.objs = append(mv.objs, obj.Id())
					newValue = false
					break
				}
			}
			if newValue {
				item.Individual = append(item.Individual, &Value[V]{val: val, objs: []uint64{obj.Id()}})
			}

			l, ok := s.cachedLengths[obj.Id()]
			if ok {
				s.cachedLengths[obj.Id()] = l + 1
			} else {
				s.cachedLengths[obj.Id()] = len(s.sharedItems) + 1
			}

			return
		}
	}

	s.appendedItems = append(s.appendedItems, &MultiValue[V]{Individual: []*Value[V]{{val: val, objs: []uint64{obj.Id()}}}})

	s.cachedLengths[obj.Id()] = s.cachedLengths[obj.Id()] + 1
}

func (s *Slice[V, O]) Detach(obj O) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for i, ind := range s.individualItems {
	individualLoop:
		for mvi, mv := range ind.Individual {
			for oi, o := range mv.objs {
				if o == obj.Id() {
					if len(mv.objs) == 1 {
						if len(ind.Individual) == 1 {
							delete(s.individualItems, i)
						} else {
							ind.Individual = append(ind.Individual[:mvi], ind.Individual[mvi+1:]...)
						}
					} else {
						mv.objs = append(mv.objs[:oi], mv.objs[oi+1:]...)
					}
					break individualLoop
				}
			}
		}
	}

appendedLoop:
	for _, item := range s.appendedItems {
		found := false
	individualLoop2:
		for mvi, mv := range item.Individual {
			for oi, o := range mv.objs {
				if o == obj.Id() {
					found = true
					if len(mv.objs) == 1 {
						item.Individual = append(item.Individual[:mvi], item.Individual[mvi+1:]...)
					} else {
						mv.objs = append(mv.objs[:oi], mv.objs[oi+1:]...)
					}
					break individualLoop2
				}
			}
		}
		if !found {
			break appendedLoop
		}
	}

	delete(s.cachedLengths, obj.Id())
}
