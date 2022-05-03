package version

const (
	Phase0 = iota
	Altair
	Bellatrix
	BellatrixBlind
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
	default:
		return "unknown version"
	}
}
