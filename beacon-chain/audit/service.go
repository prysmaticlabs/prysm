package audit

import (
	"sync"
	"time"
)

// Service is a service to provide auditing capabilities for tracking the success and failure
// of attestations
type Service struct {
	// Lock access to successCount, failureRecords
	sync.RWMutex   // Use RWMutex to allow concurrent reads
	successCount   int
	failureRecords []Failure

	successCallbacks []SuccessCallback
	failureCallbacks []FailureCallback
	resetCallbacks   []ResetCallback
}

func NewService() *Service {
	return &Service{
		failureRecords:   make([]Failure, 0),
		successCallbacks: make([]SuccessCallback, 0),
		failureCallbacks: make([]FailureCallback, 0),
		resetCallbacks:   make([]ResetCallback, 0),
	}
}

// IncrementSuccess increments the success count
func (s *Service) IncrementSuccess() {
	s.Lock()
	defer s.Unlock()

	s.successCount++

	// Trigger all registered success callbacks
	for _, callback := range s.successCallbacks {
		callback(s.successCount)
	}
}

// IncrementFailure increments the failure count and appends the reason to the failureRecords
func (s *Service) IncrementFailure(reason string) {
	s.Lock()
	defer s.Unlock()

	failure := Failure{
		Reason: reason,
		Time:   time.Now(),
	}

	// Append the reason and current time to the failureRecords slice
	s.failureRecords = append(
		s.failureRecords, failure,
	)

	// Trigger all registered failure callbacks
	for _, callback := range s.failureCallbacks {
		callback(failure)
	}
}

// Summary returns a structured summary report of the current audit state
func (s *Service) Summary() SummaryReport {
	// Using RLock to allow multiple readers to access the data
	s.RLock()
	defer s.RUnlock()

	// Return the structured summary report
	return SummaryReport{
		TotalSuccesses: s.successCount,
		TotalFailures:  len(s.failureRecords),
		Failures:       s.failureRecords,
	}
}

// Reset clears the successCount and failureRecords
func (s *Service) Reset() {
	s.Lock()
	defer s.Unlock()

	s.successCount = 0
	s.failureRecords = make([]Failure, 0)

	// Trigger all registered reset callbacks
	for _, callback := range s.resetCallbacks {
		callback()
	}
}

// RegisterSuccessCallback allows subscribers to register a callback for success events
func (s *Service) RegisterSuccessCallback(callback SuccessCallback) {
	s.Lock()
	defer s.Unlock()
	s.successCallbacks = append(s.successCallbacks, callback)
}

// RegisterFailureCallback allows subscribers to register a callback for failure events
func (s *Service) RegisterFailureCallback(callback FailureCallback) {
	s.Lock()
	defer s.Unlock()
	s.failureCallbacks = append(s.failureCallbacks, callback)
}

// RegisterResetCallback allows subscribers to register a callback for reset events
func (s *Service) RegisterResetCallback(callback ResetCallback) {
	s.Lock()
	defer s.Unlock()

	s.resetCallbacks = append(s.resetCallbacks, callback)
}
