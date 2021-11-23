package apimiddleware

type listKeystoresResponseJson struct {
	Keystores []*keystoreJson `json:"keystores"`
}

type keystoreJson struct {
	ValidatingPubkey string `json:"validating_pubkey" hex:"true"`
	DerivationPath   string `json:"derivation_path"`
}

type importKeystoresRequestJson struct {
	GenesisTime           string `json:"genesis_time" time:"true"`
	GenesisValidatorsRoot string `json:"genesis_validators_root" hex:"true"`
	GenesisForkVersion    string `json:"genesis_fork_version" hex:"true"`
}

type importKeystoresResponseJson struct {
	Statuses []*importKeystoresStatusJson `json:"statuses"`
}

type importKeystoresStatusJson struct {
	KeystorePath string `json:"keystore_path"`
	Status       string `json:"status" enum:"true"`
}

type deleteKeystoresRequestJson struct {
	PublicKeys []string `json:"public_keys" hex:"true"`
}

type deleteKeystoresStatusJson struct {
	Status  string `json:"status" enum:"true"`
	Message string `json:"message"`
}

type deleteKeystoresResponseJson struct {
	Statuses           []*deleteKeystoresStatusJson `json:"statuses"`
	SlashingProtection string                       `json:"slashing_protection"`
}
