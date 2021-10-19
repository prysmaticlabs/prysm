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
	case Merge:
		return "merge"
	default:
		return "unknown version"
	}
}
