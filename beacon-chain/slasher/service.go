// Package slasher --
package slasher

import (
	"context"
)

// Service --
type Service struct {
}

// NewService --
func NewService(ctx context.Context) (*Service, error) {
	return &Service{}, nil
}

// Start --
func (s *Service) Start() {
}

// Stop --
func (s *Service) Stop() error {
	return nil
}

// Status --
func (s *Service) Status() error {
	return nil
}
