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

type statusRemoteKeysJson struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type deleteRemoteKeysResponseJson struct {
	Statuses []*statusJson `json:"data"`
}
