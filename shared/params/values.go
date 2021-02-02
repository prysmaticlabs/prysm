package params

const (
	Mainnet configName = iota
	EndToEnd
	Pyrmont
	Toledo
)

// ConfigNames provides network configuration names.
var ConfigNames = map[configName]string{
	Mainnet:  "mainnet",
	EndToEnd: "end-to-end",
	Pyrmont:  "pyrmont",
	Toledo:   "toledo",
}

type configName = int
