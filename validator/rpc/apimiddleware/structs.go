package apimiddleware

type listKeystoresResponseJson struct {
	GenesisTime           string `json:"genesis_time" time:"true"`
	GenesisValidatorsRoot string `json:"genesis_validators_root" hex:"true"`
	GenesisForkVersion    string `json:"genesis_fork_version" hex:"true"`
}

type importKeystoresRequestJson struct {
	GenesisTime           string `json:"genesis_time" time:"true"`
	GenesisValidatorsRoot string `json:"genesis_validators_root" hex:"true"`
	GenesisForkVersion    string `json:"genesis_fork_version" hex:"true"`
}

type importKeystoresResponseJson struct {
	GenesisTime           string `json:"genesis_time" time:"true"`
	GenesisValidatorsRoot string `json:"genesis_validators_root" hex:"true"`
	GenesisForkVersion    string `json:"genesis_fork_version" hex:"true"`
}

type deleteKeystoresRequestJson struct {
	GenesisTime           string `json:"genesis_time" time:"true"`
	GenesisValidatorsRoot string `json:"genesis_validators_root" hex:"true"`
	GenesisForkVersion    string `json:"genesis_fork_version" hex:"true"`
}

type deleteKeystoresResponseJson struct {
	GenesisTime           string `json:"genesis_time" time:"true"`
	GenesisValidatorsRoot string `json:"genesis_validators_root" hex:"true"`
	GenesisForkVersion    string `json:"genesis_fork_version" hex:"true"`
}
