package version

const (
	Phase0 = iota
	Altair
	Merge
)

func String(version int) string {
	switch version {
	case Phase0:
		return "phase0"
	case Altair:
		return "altair"
	default:
		return "unknown version"
	}
}
