package version

const (
	Phase0 = iota
	Altair
	Bellatrix
	Capella
	EIP4844
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
	case EIP4844:
		return "eip4844"
	default:
		return "unknown version"
	}
}

// All returns a list of all known fork versions.
func All() []int {
	return []int{Phase0, Altair, Bellatrix, Capella}
}
