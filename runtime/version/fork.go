package version

const (
	Phase0 = iota
	Altair
	Bellatrix
	Capella
	Deneb
)

func String(version int) string {
	switch version {
	case Phase0:
		return "phase0"
	case Altair:
		return "altair"
	case Bellatrix:
		return "bellatrix"
	case Capella:
		return "capella"
	case Deneb:
		return "deneb"
	default:
		return "unknown version"
	}
}

// All returns a list of all known fork versions.
func All() []int {
	return []int{Phase0, Altair, Bellatrix, Capella}
}
