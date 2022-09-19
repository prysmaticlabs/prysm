package version

const (
	Phase0 = iota
	Altair
	Bellatrix
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
	case EIP4844:
		return "eip4844"
	default:
		return "unknown version"
	}
}
