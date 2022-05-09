package params

var Registry *registry

type registry struct {
}

func init() {
	Registry = &registry{}
}
