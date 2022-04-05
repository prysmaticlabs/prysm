package apimiddleware

type listKeystoresResponseJson struct {
	Keystores []*keystoreJson `json:"data"`
}

type keystoreJson struct {
	ValidatingPubkey string `json:"validating_pubkey" hex:"true"`
	DerivationPath   string `json:"derivation_path"`
}

type importKeystoresRequestJson struct {
	Keystores          []string `json:"keystores"`
	Passwords          []string `json:"passwords"`
	SlashingProtection string   `json:"slashing_protection"`
}

type importKeystoresResponseJson struct {
	Statuses []*statusJson `json:"data"`
}

type deleteKeystoresRequestJson struct {
	PublicKeys []string `json:"pubkeys" hex:"true"`
}

type statusJson struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type deleteKeystoresResponseJson struct {
	Statuses           []*statusJson `json:"data"`
	SlashingProtection string        `json:"slashing_protection"`
}
