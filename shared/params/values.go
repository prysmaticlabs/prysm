package params

const (
	Mainnet ConfigName = iota
	EndToEnd
	Pyrmont
	Toledo
	Prater
)

// ConfigNames provides network configuration names.
var ConfigNames = map[ConfigName]string{
	Mainnet:  "mainnet",
	EndToEnd: "end-to-end",
	Pyrmont:  "pyrmont",
	Toledo:   "toledo",
	Prater:   "prater",
}

// ConfigName enum describes the type of known network in use.
type ConfigName = int
