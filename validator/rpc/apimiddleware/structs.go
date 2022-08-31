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
	Status  string `json:"status" enum:"true"`
	Message string `json:"message"`
}

type deleteKeystoresResponseJson struct {
	Statuses           []*statusJson `json:"data"`
	SlashingProtection string        `json:"slashing_protection"`
}

//remote keymanager api

type listRemoteKeysResponseJson struct {
	Keystores []*remoteKeysListJson `json:"data"`
}

type remoteKeysListJson struct {
	Pubkey   string `json:"pubkey" hex:"true"`
	Url      string `json:"url"`
	Readonly bool   `json:"readonly"`
}

type remoteKeysJson struct {
	Pubkey   string `json:"pubkey" hex:"true"`
	Url      string `json:"url"`
	Readonly bool   `json:"readonly"`
}

type importRemoteKeysRequestJson struct {
	Keystores []*remoteKeysJson `json:"remote_keys"`
}

type importRemoteKeysResponseJson struct {
	Statuses []*statusJson `json:"data"`
}

type deleteRemoteKeysRequestJson struct {
	PublicKeys []string `json:"pubkeys" hex:"true"`
}

type deleteRemoteKeysResponseJson struct {
	Statuses []*statusJson `json:"data"`
}

type feeRecipientJson struct {
	Pubkey     string `json:"pubkey" hex:"true"`
	Ethaddress string `json:"ethaddress" address:"true"`
}

type gasLimitJson struct {
	Pubkey   string `json:"pubkey" hex:"true"`
	GasLimit string `json:"gas_limit"`
}

type getFeeRecipientByPubkeyResponseJson struct {
	Data *feeRecipientJson `json:"data"`
}

type setFeeRecipientByPubkeyRequestJson struct {
	Ethaddress string `json:"ethaddress" hex:"true"`
}

type getGasLimitResponseJson struct {
	Data *gasLimitJson `json:"data"`
}

type setGasLimitRequestJson struct {
	GasLimit string `json:"gas_limit"`
}

type deleteGasLimitRequestJson struct {
	Pubkey string `json:"pubkey" hex:"true"`
}
