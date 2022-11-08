// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package queue

// Queue of an arbitrary type backed by a slice which is shrinked when it grows too large
type Queue[T any] struct {
	slice []T
}

func (q *Queue[T]) Push(item T) {
	q.slice = append(q.slice, item)
}

// If cap(slice) >= len(slice)*shrinkRatio && cap(slice) >= shrinkMinimum,
// shrink the slice capacity back down to twice its length by re-allocating it.
const shrinkRatio = 16
const shrinkMinimum = 512

func (q *Queue[T]) shrink() {
	if cap(q.slice) >= len(q.slice)*shrinkRatio && cap(q.slice) >= shrinkMinimum {
		oldSlice := q.slice
		q.slice = make([]T, len(oldSlice), len(oldSlice)*2)
		copy(q.slice, oldSlice)
	}
}

func (q *Queue[T]) Pop() T {
	var empty T
	if len(q.slice) == 0 {
		return empty
	}
	item := q.slice[0]
	q.slice[0] = empty
	q.slice = q.slice[1:]
	q.shrink()
	return item
}

func (q *Queue[T]) Len() int {
	return len(q.slice)
}

// Peek returns n items from the queue without removing them.
// If n is too large, it returns all queued items.
func (q *Queue[T]) Peek(n int) []T {
	if n >= q.Len() {
		return q.slice
	}
	return q.slice[q.Len()-n:]
}
