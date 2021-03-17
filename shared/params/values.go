package params

const (
	Mainnet configName = iota
	EndToEnd
	Pyrmont
	Toledo
	Prater
)

// ConfigNames provides network configuration names.
var ConfigNames = map[configName]string{
	Mainnet:  "mainnet",
	EndToEnd: "end-to-end",
	Pyrmont:  "pyrmont",
	Toledo:   "toledo",
	Prater:   "prater",
}

type configName = int
