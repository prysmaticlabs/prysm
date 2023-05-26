package multi_value_slice

import (
	"fmt"
	"sync"
)

type Orderable interface {
	Order() uint64
}

type MultiValueSlice interface {
	Len() int
}

type Value[V any] struct {
	val  V
	objs []uint64
}

type MultiValue[V any] struct {
	Shared     V
	Individual []*Value[V]
}

type Slice[V comparable, O Orderable] struct {
	Items []*MultiValue[V]
	lock  sync.RWMutex
}

func (s *Slice[V, O]) Len() int {
	return len(s.Items)
}

func (s *Slice[V, O]) Copy(src O, dst O) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, item := range s.Items {
		if item.Individual != nil {
		outerLoop:
			for _, mv := range item.Individual {
				for _, o := range mv.objs {
					if o == src.Order() {
						mv.objs = append(mv.objs, dst.Order())
						break outerLoop
					}
				}
			}
		}
	}
}

func (s *Slice[V, O]) Value(obj O) []V {
	s.lock.RLock()
	defer s.lock.RUnlock()

	v := make([]V, len(s.Items))
	for i, item := range s.Items {
		if item.Individual == nil {
			v[i] = s.Items[i].Shared
		} else {
			found := false
		outerLoop:
			for _, mv := range item.Individual {
				for _, o := range mv.objs {
					if o == obj.Order() {
						v[i] = mv.val
						found = true
						break outerLoop
					}
				}
			}
			if !found {
				v[i] = s.Items[i].Shared
			}
		}
	}

	return v
}

func (s *Slice[V, O]) At(obj O, i uint64) (V, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if i >= uint64(len(s.Items)) {
		var def V
		return def, fmt.Errorf("index %d is out of bounds", i)
	}

	item := s.Items[i]
	if item.Individual == nil {
		return item.Shared, nil
	}
	for _, mv := range item.Individual {
		for _, o := range mv.objs {
			if o == obj.Order() {
				return mv.val, nil
			}
		}
	}
	return item.Shared, nil
}

func (s *Slice[V, O]) UpdateAt(obj O, i uint64, val V) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if i >= uint64(len(s.Items)) {
		return fmt.Errorf("index %d is out of bounds", i)
	}

	item := s.Items[i]

outerLoop:
	for mvi, mv := range item.Individual {
		for oi, o := range mv.objs {
			if o == obj.Order() {
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
			mv.objs = append(mv.objs, obj.Order())
			newValue = false
			break
		}
	}
	if newValue {
		item.Individual = append(item.Individual, &Value[V]{val: val, objs: []uint64{obj.Order()}})
	}

	return nil
}

func (s *Slice[V, O]) Detach(obj O) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, item := range s.Items {
	outerLoop:
		for mvi, mv := range item.Individual {
			for oi, o := range mv.objs {
				if o == obj.Order() {
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
