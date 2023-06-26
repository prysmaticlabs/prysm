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
	SharedItems     []V
	IndividualItems map[Id]*MultiValue[V]
	AppendedItems   []*MultiValue[V]
	lock            sync.RWMutex
}

func (s *Slice[V, O]) Len(obj O) int {
	s.lock.RLock()
	defer s.lock.RUnlock()

	l := len(s.SharedItems)
	for i := 0; i < len(s.AppendedItems); i++ {
		found := false
		for _, mv := range s.AppendedItems[i].Individual {
			for _, o := range mv.objs {
				if o == obj.Id() {
					found = true
					l++
				}
			}
		}
		if !found {
			return l
		}
	}
	return l
}

func (s *Slice[V, O]) Copy(src O, dst O) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, item := range s.IndividualItems {
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

	for _, item := range s.AppendedItems {
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
}

func (s *Slice[V, O]) Value(obj O) []V {
	s.lock.RLock()
	defer s.lock.RUnlock()

	v := make([]V, len(s.SharedItems))
	for i, item := range s.SharedItems {
		ind, ok := s.IndividualItems[uint64(i)]
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

	for _, item := range s.AppendedItems {
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

	if i >= uint64(len(s.SharedItems)+len(s.AppendedItems)) {
		var def V
		return def, fmt.Errorf("index %d is out of bounds", i)
	}

	isOriginal := i < uint64(len(s.SharedItems))
	if isOriginal {
		ind, ok := s.IndividualItems[i]
		if !ok {
			return s.SharedItems[i], nil
		}
		for _, mv := range ind.Individual {
			for _, o := range mv.objs {
				if o == obj.Id() {
					return mv.val, nil
				}
			}
		}
		return s.SharedItems[i], nil
	} else {
		item := s.AppendedItems[i-uint64(len(s.SharedItems))]
		for _, mv := range item.Individual {
			for _, o := range mv.objs {
				if o == obj.Id() {
					return mv.val, nil
				}
			}
		}
		var def V
		return def, fmt.Errorf("index %d is out of bounds", i)
	}
}

func (s *Slice[V, O]) UpdateAt(obj O, i uint64, val V) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if i >= uint64(len(s.SharedItems)+len(s.AppendedItems)) {
		return fmt.Errorf("index %d is out of bounds", i)
	}

	isOriginal := i < uint64(len(s.SharedItems))
	if isOriginal {
		ind, ok := s.IndividualItems[i]
		if ok {
		outerLoop:
			for mvi, mv := range ind.Individual {
				for oi, o := range mv.objs {
					if o == obj.Id() {
						if len(mv.objs) == 1 {
							// TODO: Can we delete this safely?
							//if len(ind.Individual) == 1 {
							//delete(s.IndividualItems, i)
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

		if val == s.SharedItems[i] {
			return nil
		}

		if !ok {
			s.IndividualItems[i] = &MultiValue[V]{Individual: []*Value[V]{{val: val, objs: []uint64{obj.Id()}}}}
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
		item := s.AppendedItems[i-uint64(len(s.SharedItems))]
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
			return fmt.Errorf("index %d is out of bounds", i)
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

	if len(s.AppendedItems) == 0 {
		s.AppendedItems = []*MultiValue[V]{{Individual: []*Value[V]{{val: val, objs: []uint64{obj.Id()}}}}}
		return
	}

	for _, item := range s.AppendedItems {
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

	s.AppendedItems = append(s.AppendedItems, &MultiValue[V]{Individual: []*Value[V]{{val: val, objs: []uint64{obj.Id()}}}})
}

func (s *Slice[V, O]) Detach(obj O) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for i, ind := range s.IndividualItems {
	outerLoop:
		for mvi, mv := range ind.Individual {
			for oi, o := range mv.objs {
				if o == obj.Id() {
					if len(mv.objs) == 1 {
						if len(ind.Individual) == 1 {
							delete(s.IndividualItems, i)
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
	for _, item := range s.AppendedItems {
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
}
