package interfaces

// Id is an object identifier.
type Id = uint64

// Identifiable represents an object that can be uniquely identified by its Id.
type Identifiable interface {
	Id() Id
	SetId(id uint64)
}
