package params

const (
	Mainnet ConfigName = iota
	EndToEnd
	Pyrmont
	Toledo
	Prater
	L15
)

// ConfigNames provides network configuration names.
var ConfigNames = map[ConfigName]string{
	Mainnet:  "mainnet",
	EndToEnd: "end-to-end",
	Pyrmont:  "pyrmont",
	Toledo:   "toledo",
	Prater:   "prater",
	L15:      "lukso-private-testnet",
}

// ConfigName enum describes the type of known network in use.
type ConfigName = int
