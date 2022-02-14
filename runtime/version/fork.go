package version

const (
	Phase0 = iota
	Altair
	Bellatrix
	Sharding
)

func String(version int) string {
	switch version {
	case Phase0:
		return "phase0"
	case Altair:
		return "altair"
	case Bellatrix:
		return "bellatrix"
	case Sharding:
		return "sharding"
	default:
		return "unknown version"
	}
}
