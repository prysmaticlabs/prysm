package attestations

import (
    "fmt"
    "sync"

    "github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "pool/attestations")

// AttestationStats tracks successfully verified and failed attestations
type AttestationStats struct {
    TotalVerified  int            // Number of successfully verified attestations
    TotalFailed    int            // Number of failed attestations
    FailureReasons map[string]int // A map of failure reasons and their counts
    mu             sync.Mutex     // Ensures safe concurrent access
}

// NewAttestationStats creates a new instance of AttestationStats
func NewAttestationStats() *AttestationStats {
    return &AttestationStats{
        FailureReasons: make(map[string]int),
    }
}

// LogSuccess increments the count of successfully verified attestations
func (s *AttestationStats) LogSuccess() {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.TotalVerified++
    log.WithField("verified", s.TotalVerified).Info("Verified attestation")
}

// LogFailure increments the count of failed attestations and records the reason
func (s *AttestationStats) LogFailure(reason string) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.TotalFailed++
    s.FailureReasons[reason]++
    log.WithFields(logrus.Fields{
        "reason": reason,
        "total_failed": s.TotalFailed,
    }).Warn("Failed attestation recorded")
}

// Reset clears the stats at the end of each epoch
func (s *AttestationStats) Reset() {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.TotalVerified = 0
    s.TotalFailed = 0
    s.FailureReasons = make(map[string]int)
    log.Info("Attestation stats reset for new epoch")
}

// Summary outputs a summary of the collected attestation stats at the end of each epoch
func (s *AttestationStats) Summary() string {
    s.mu.Lock()
    defer s.mu.Unlock()
    summary := fmt.Sprintf("Epoch Summary: Total Verified: %d, Total Failed: %d, Failure Reasons: %v",
        s.TotalVerified, s.TotalFailed, s.FailureReasons)
    log.Info(summary)
    return summary
}