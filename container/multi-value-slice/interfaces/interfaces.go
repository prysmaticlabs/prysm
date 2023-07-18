package interfaces

import "github.com/google/uuid"

// Identifiable represents an object that can be uniquely identified by its Id.
type Identifiable interface {
	Id() uuid.UUID
	SetId(id uuid.UUID)
}
