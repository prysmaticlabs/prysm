package mputil

import (
	"errors"
	"runtime"
	"sync"
)

// Batch provides details of the scatter process
type Batch struct {
	Workers  int
	ResultCh chan *WorkerResults
	ErrorCh  chan error
}

// Scatter scatters a computation across multiple goroutines.
// This breaks the task in to a number of chunks and executes those chunks in parallel with the function provided.
// Results from the function are returned in the results channel, errors in the error channel.
// In total the number of items in the results plus error channel will equal the number of workers.
//
// Scatter returns the results of the workers; it is up to the calling code to piece them together.
func Scatter(inputLen int, sFunc func(int, int, *sync.Mutex) (interface{}, error)) ([]*WorkerResults, error) {
	if inputLen <= 0 {
		return nil, errors.New("input length must be greater than 0")
	}

	chunkSize := calculateChunkSize(inputLen)
	workers := inputLen / chunkSize
	if inputLen%chunkSize != 0 {
		workers++
	}
	resultCh := make(chan *WorkerResults, workers)
	defer close(resultCh)
	errorCh := make(chan error, workers)
	defer close(errorCh)
	mutex := new(sync.Mutex)
	for worker := 0; worker < workers; worker++ {
		offset := worker * chunkSize
		entries := chunkSize
		if offset+entries > inputLen {
			entries = inputLen - offset
		}
		go func(offset int, entries int) {
			extent, err := sFunc(offset, entries, mutex)
			if err != nil {
				errorCh <- err
			} else {
				resultCh <- &WorkerResults{
					Offset: offset,
					Extent: extent,
				}
			}
		}(offset, entries)
	}

	// Gather
	results := make([]*WorkerResults, workers)
	for i := 0; i < workers; i++ {
		select {
		case result := <-resultCh:
			results[i] = result
		case err := <-errorCh:
			return nil, err
		}
	}
	return results, nil
}

// calculateChunkSize calculates a suitable chunk size for the purposes of parallelisation.
func calculateChunkSize(items int) int {
	// Start with a simple even split
	chunkSize := items / runtime.GOMAXPROCS(0)

	// Add 1 if we have leftovers (or if we have fewer items than processors)
	if chunkSize == 0 || items%chunkSize != 0 {
		chunkSize++
	}

	return chunkSize
}

// WorkerResults are the results of a scatter worker
type WorkerResults struct {
	Offset int
	Extent interface{}
}

// NewWorkerResults creates a new container for results of a scatter worker
func NewWorkerResults(offset int, extent interface{}) *WorkerResults {
	return &WorkerResults{
		Offset: offset,
		Extent: extent,
	}
}
