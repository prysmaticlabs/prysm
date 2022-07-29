package version

const (
	Phase0 = iota
	Altair
	Bellatrix
	BellatrixBlind
	Capella
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
	case Capella:
		return "capella"
	default:
		return "unknown version"
	}
}
