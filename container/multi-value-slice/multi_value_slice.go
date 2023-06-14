package multi_value_slice

import (
	"fmt"
	"sync"
)

type Identifiable interface {
	Id() uint64
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

type ShareableMultiValue[V any] struct {
	Shared     V
	Individual []*Value[V]
}

type Slice[V comparable, O Identifiable] struct {
	OriginalItems []*ShareableMultiValue[V]
	AppendedItems []*MultiValue[V]
	lock          sync.RWMutex
}

func (s *Slice[V, O]) Len(obj O) int {
	l := len(s.OriginalItems)
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

	for _, item := range s.OriginalItems {
		if item.Individual != nil {
		outerLoop1:
			for _, mv := range item.Individual {
				for _, o := range mv.objs {
					if o == src.Id() {
						mv.objs = append(mv.objs, dst.Id())
						break outerLoop1
					}
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

	v := make([]V, len(s.OriginalItems))
	for i, item := range s.OriginalItems {
		if item.Individual == nil {
			v[i] = s.OriginalItems[i].Shared
		} else {
			found := false
		outerLoop1:
			for _, mv := range item.Individual {
				for _, o := range mv.objs {
					if o == obj.Id() {
						v[i] = mv.val
						found = true
						break outerLoop1
					}
				}
			}
			if !found {
				v[i] = s.OriginalItems[i].Shared
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

	if i >= uint64(len(s.OriginalItems)+len(s.AppendedItems)) {
		var def V
		return def, fmt.Errorf("index %d is out of bounds", i)
	}

	isOriginal := i < uint64(len(s.OriginalItems))
	if isOriginal {
		item := s.OriginalItems[i]
		if item.Individual == nil {
			return item.Shared, nil
		}
		for _, mv := range item.Individual {
			for _, o := range mv.objs {
				if o == obj.Id() {
					return mv.val, nil
				}
			}
		}
		return item.Shared, nil
	} else {
		item := s.AppendedItems[i-uint64(len(s.OriginalItems))]
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

	if i >= uint64(len(s.OriginalItems)+len(s.AppendedItems)) {
		return fmt.Errorf("index %d is out of bounds", i)
	}

	isOriginal := i < uint64(len(s.OriginalItems))
	if isOriginal {
		item := s.OriginalItems[i]
	outerLoop1:
		for mvi, mv := range item.Individual {
			for oi, o := range mv.objs {
				if o == obj.Id() {
					if len(mv.objs) == 1 {
						item.Individual = append(item.Individual[:mvi], item.Individual[mvi+1:]...)
					} else {
						mv.objs = append(mv.objs[:oi], mv.objs[oi+1:]...)
					}
					break outerLoop1
				}
			}
		}

		if val == item.Shared {
			return nil
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
	} else {
		item := s.AppendedItems[i-uint64(len(s.OriginalItems))]
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

	for _, item := range s.OriginalItems {
	outerLoop1:
		for mvi, mv := range item.Individual {
			for oi, o := range mv.objs {
				if o == obj.Id() {
					if len(mv.objs) == 1 {
						item.Individual = append(item.Individual[:mvi], item.Individual[mvi+1:]...)
					} else {
						mv.objs = append(mv.objs[:oi], mv.objs[oi+1:]...)
					}
					break outerLoop1
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
