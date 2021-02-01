package params

const (
	Mainnet configName = iota
	EndToEnd
	Pyrmont
	Toledo
)

// ConfigNames provides network configuration names.
var ConfigNames = map[configName]string{
	Mainnet:  "Mainnet",
	EndToEnd: "End-to-end",
	Pyrmont:  "pyrmont",
	Toledo:   "toledo",
}

type configName = int
