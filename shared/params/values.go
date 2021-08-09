package params

const (
	Mainnet ConfigName = iota
	Minimal
	EndToEnd
	Pyrmont
	Toledo
	Prater
)

// ConfigNames provides network configuration names.
var ConfigNames = map[ConfigName]string{
	Mainnet:  "mainnet",
	Minimal:  "minimal",
	EndToEnd: "end-to-end",
	Pyrmont:  "pyrmont",
	Toledo:   "toledo",
	Prater:   "prater",
}

// ConfigName enum describes the type of known network in use.
type ConfigName = int
