package multi_value_slice

import (
	"fmt"
	"sync"
)

type Id = uint64

type Identifiable interface {
	Id() Id
	SetId(id uint64)
}

type MultiValueSlice[O Identifiable] interface {
	Len(obj O) int
}

type Value[V any] struct {
	val  V
	objs []uint64
}

type MultiValue[V any] struct {
	Individual []*Value[V]
}

type Slice[V comparable, O Identifiable] struct {
	sharedItems     []V
	individualItems map[Id]*MultiValue[V]
	appendedItems   []*MultiValue[V]
	lengths         map[Id]int
	lock            sync.RWMutex
}

func (s *Slice[V, O]) Init(items []V) {
	s.sharedItems = items
	s.individualItems = map[Id]*MultiValue[V]{}
	s.appendedItems = []*MultiValue[V]{}
	s.lengths = map[Id]int{}
}

func (s *Slice[V, O]) Len(obj O) int {
	s.lock.RLock()
	defer s.lock.RUnlock()

	l, ok := s.lengths[obj.Id()]
	if !ok {
		return len(s.sharedItems)
	}
	return l
}

func (s *Slice[V, O]) Copy(src O, dst O) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, item := range s.individualItems {
	outerLoop:
		for _, mv := range item.Individual {
			for _, o := range mv.objs {
				if o == src.Id() {
					mv.objs = append(mv.objs, dst.Id())
					break outerLoop
				}
			}
		}
	}

	for _, item := range s.appendedItems {
		found := false
	outerLoop2:
		for _, mv := range item.Individual {
			for _, o := range mv.objs {
				if o == src.Id() {
					found = true
					mv.objs = append(mv.objs, dst.Id())
					break outerLoop2
				}
			}
		}
		if !found {
			return
		}
	}

	srcLen, ok := s.lengths[src.Id()]
	if ok {
		s.lengths[dst.Id()] = srcLen
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
		outerLoop:
			for _, mv := range ind.Individual {
				for _, o := range mv.objs {
					if o == obj.Id() {
						v[i] = mv.val
						found = true
						break outerLoop
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
	outerLoop2:
		for _, mv := range item.Individual {
			for _, o := range mv.objs {
				if o == obj.Id() {
					found = true
					v = append(v, mv.val)
					break outerLoop2
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
		outerLoop:
			for mvi, mv := range ind.Individual {
				for oi, o := range mv.objs {
					if o == obj.Id() {
						if len(mv.objs) == 1 {
							// TODO: Can we delete this safely?
							//if len(ind.Individual) == 1 {
							//delete(s.individualItems, i)
							//} else {
							ind.Individual = append(ind.Individual[:mvi], ind.Individual[mvi+1:]...)
							//}
						} else {
							mv.objs = append(mv.objs[:oi], mv.objs[oi+1:]...)
						}
						break outerLoop
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
	outerLoop2:
		for mvi, mv := range item.Individual {
			for oi, o := range mv.objs {
				if o == obj.Id() {
					found = true
					if len(mv.objs) == 1 {
						item.Individual = append(item.Individual[:mvi], item.Individual[mvi+1:]...)
					} else {
						mv.objs = append(mv.objs[:oi], mv.objs[oi+1:]...)
					}
					break outerLoop2
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
		s.appendedItems = []*MultiValue[V]{{Individual: []*Value[V]{{val: val, objs: []uint64{obj.Id()}}}}}
		return
	}

	for _, item := range s.appendedItems {
		found := false
	outerLoop:
		for _, mv := range item.Individual {
			for _, o := range mv.objs {
				if o == obj.Id() {
					found = true
					break outerLoop
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
			return
		}
	}

	s.appendedItems = append(s.appendedItems, &MultiValue[V]{Individual: []*Value[V]{{val: val, objs: []uint64{obj.Id()}}}})

	srcLen, ok := s.lengths[obj.Id()]
	if ok {
		s.lengths[obj.Id()] = srcLen + 1
	} else {
		s.lengths[obj.Id()] = len(s.sharedItems) + 1
	}
}

func (s *Slice[V, O]) Detach(obj O) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for i, ind := range s.individualItems {
	outerLoop:
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
					break outerLoop
				}
			}
		}
	}
	for _, item := range s.appendedItems {
		found := false
	outerLoop2:
		for mvi, mv := range item.Individual {
			for oi, o := range mv.objs {
				if o == obj.Id() {
					found = true
					if len(mv.objs) == 1 {
						item.Individual = append(item.Individual[:mvi], item.Individual[mvi+1:]...)
					} else {
						mv.objs = append(mv.objs[:oi], mv.objs[oi+1:]...)
					}
					break outerLoop2
				}
			}
		}
		if !found {
			return
		}
	}

	delete(s.lengths, obj.Id())
}
