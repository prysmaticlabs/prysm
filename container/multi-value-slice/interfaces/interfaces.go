package interfaces

type Id = uint64

type Identifiable interface {
	Id() Id
	SetId(id uint64)
}
