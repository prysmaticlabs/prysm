package blockchain

import (
	"fmt"
	"sync"
)

type attestationMetrics struct {
	successCount   int
	failureCount   int
	failureReasons map[string]int
	mu             sync.Mutex
}

func (a *attestationMetrics) AddSuccess() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.successCount++
}

func (a *attestationMetrics) AddFailure(reason string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.failureCount++
	a.failureReasons[reason]++
}

func (a *attestationMetrics) Summary() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	summary := fmt.Sprintf("Successes: %d, Failures: %d\n", a.successCount, a.failureCount)
	for reason, count := range a.failureReasons {
		summary += fmt.Sprintf("Failure Reason: %s, Count: %d\n", reason, count)
	}
	return summary
}

func (a *attestationMetrics) ReInit() {
	a.failureReasons = make(map[string]int)
}
