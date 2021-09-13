package params

const (
	Mainnet ConfigName = iota
	Minimal
	EndToEnd
	Pyrmont
	Prater
)

// ConfigNames provides network configuration names.
var ConfigNames = map[ConfigName]string{
	Mainnet:  "mainnet",
	Minimal:  "minimal",
	EndToEnd: "end-to-end",
	Pyrmont:  "pyrmont",
	Prater:   "prater",
}

// ConfigName enum describes the type of known network in use.
type ConfigName = int
