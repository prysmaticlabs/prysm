package version

const (
	Phase0 = iota
	Altair
	Bellatrix
	BellatrixBlind
	Eip4844
)

func String(version int) string {
	switch version {
	case Phase0:
		return "phase0"
	case Altair:
		return "altair"
	case Bellatrix:
		return "bellatrix"
	case BellatrixBlind:
		return "bellatrix-blind"
	case Eip4844:
		return "eip-4844"
	default:
		return "unknown version"
	}
}
