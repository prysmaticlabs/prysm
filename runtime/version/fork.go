package version

const (
	Phase0 = iota
	Altair
	Merge // TODO: subject to change, community is deciding on a star name: https://ethereum-magicians.org/t/ethereum-roadmapping-improvements/6653/14
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
