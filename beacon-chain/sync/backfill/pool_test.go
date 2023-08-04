package backfill

import "github.com/pkg/errors"

type mockPool struct {
	todo         []batch
	finished     []batch
	spawnCalled  []int
	finishedChan chan batch
	todoChan     chan batch
}

func (m *mockPool) Spawn(n int) {
}

func (m *mockPool) todoLeftPop() (batch, error) {
	if len(m.todo) == 0 {
		return batch{}, errors.New("can't pop empty todo")
	}
	b := m.todo[0]
	m.todo = m.todo[1:]
	return b, nil
}

func (m *mockPool) Todo(b batch) {
	if m.todoChan != nil {
		m.todoChan <- b
		return
	}
	m.todo = append(m.todo, b)
	return
}

func (m *mockPool) Finished() (batch, error) {
	if m.finishedChan != nil {
		return <-m.finishedChan, nil
	}
	b := m.finished[0]
	m.finished = m.finished[1:]
	return b, nil
}

var _ BatchWorkerPool = &mockPool{}
