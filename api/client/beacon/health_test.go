package beacon

import (
	"sync"
	"testing"
)

func TestNodeHealth_IsHealthy(t *testing.T) {
	tests := []struct {
		name      string
		isHealthy bool
		want      bool
	}{
		{"initially healthy", true, true},
		{"initially unhealthy", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &NodeHealth{
				isHealthy: tt.isHealthy,
				HealthCh:  make(chan bool, 1),
			}
			if got := n.IsHealthy(); got != tt.want {
				t.Errorf("IsHealthy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeHealth_UpdateNodeHealth(t *testing.T) {
	tests := []struct {
		name       string
		initial    bool // Initial health status
		newStatus  bool // Status to update to
		shouldSend bool // Should a message be sent through the channel
	}{
		{"healthy to unhealthy", true, false, true},
		{"unhealthy to healthy", false, true, true},
		{"remain healthy", true, true, false},
		{"remain unhealthy", false, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := NewNodeHealth()
			n.isHealthy = tt.initial // Set initial health status
			n.UpdateNodeHealth(tt.newStatus)

			// Check if health status was updated
			if n.IsHealthy() != tt.newStatus {
				t.Errorf("UpdateNodeHealth() failed to update isHealthy from %v to %v", tt.initial, tt.newStatus)
			}

			select {
			case status := <-n.HealthCh:
				if !tt.shouldSend {
					t.Errorf("UpdateNodeHealth() unexpectedly sent status %v to HealthCh", status)
				} else if status != tt.newStatus {
					t.Errorf("UpdateNodeHealth() sent wrong status %v, want %v", status, tt.newStatus)
				}
			default:
				if tt.shouldSend {
					t.Error("UpdateNodeHealth() did not send any status to HealthCh when expected")
				}
			}
		})
	}
}

func TestNodeHealth_Concurrency(t *testing.T) {
	n := NewNodeHealth()
	var wg sync.WaitGroup

	// Number of goroutines to spawn for both reading and writing
	numGoroutines := 6

	go func() {
		for range n.HealthCh {
			// Consume values to avoid blocking on channel send.
		}
	}()

	wg.Add(numGoroutines * 2) // for readers and writers

	// Concurrently update health status
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			n.UpdateNodeHealth(false) // Flip health status
			n.UpdateNodeHealth(true)  // And flip it back
		}()
	}

	// Concurrently read health status
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_ = n.IsHealthy() // Just read the value
		}()
	}

	wg.Wait() // Wait for all goroutines to finish
}
