package validator_service_config

import (
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
)

// ProposerSettingsPayload is the struct representation of the JSON or YAML payload set in the validator through the CLI.
// ProposerConfig is the map of validator address to fee recipient options all in hex format.
// DefaultConfig is the default fee recipient address for all validators unless otherwise specified in the propose config.required.
type ProposerSettingsPayload struct {
	ProposerConfig map[string]*ProposerOptionPayload `json:"proposer_config" yaml:"proposer_config"`
	DefaultConfig  *ProposerOptionPayload            `json:"default_config" yaml:"default_config"`
}

// ToSettings converts struct to ProposerSettings
func (ps *ProposerSettingsPayload) ToSettings() (*ProposerSettings, error) {
	if ps.DefaultConfig == nil || ps.DefaultConfig.FeeRecipient == "" {
		return nil, errors.New("payload default config is missing or default fee recipient is missing")
	}
	settings := &ProposerSettings{}
	if ps.ProposerConfig != nil {
		settings.ProposeConfig = make(map[[fieldparams.BLSPubkeyLength]byte]*ProposerOption)
		for key, optionPayload := range ps.ProposerConfig {
			b, err := hexutil.Decode(key)
			if err != nil {
				return nil, err
			}
			settings.ProposeConfig[bytesutil.ToBytes48(b)] = &ProposerOption{
				FeeRecipientConfig: &FeeRecipientConfig{
					FeeRecipient: common.HexToAddress(optionPayload.FeeRecipient),
				},
				BuilderConfig: optionPayload.BuilderConfig.Clone(),
			}
		}
	}
	settings.DefaultConfig = &ProposerOption{
		FeeRecipientConfig: &FeeRecipientConfig{
			FeeRecipient: common.HexToAddress(ps.DefaultConfig.FeeRecipient),
		},
		BuilderConfig: ps.DefaultConfig.BuilderConfig.Clone(),
	}
	return settings, nil
}

// ProposerOptionPayload is the struct representation of the JSON config file set in the validator through the CLI.
// FeeRecipient is set to an eth address in hex string format with 0x prefix.
type ProposerOptionPayload struct {
	FeeRecipient  string         `json:"fee_recipient" yaml:"fee_recipient"`
	BuilderConfig *BuilderConfig `json:"builder" yaml:"builder"`
}

// BuilderConfig is the struct representation of the JSON config file set in the validator through the CLI.
// GasLimit is a number set to help the network decide on the maximum gas in each block.
type BuilderConfig struct {
	Enabled  bool     `json:"enabled" yaml:"enabled"`
	GasLimit Uint64   `json:"gas_limit,omitempty" yaml:"gas_limit,omitempty"`
	Relays   []string `json:"relays" yaml:"relays"`
}

// Uint64 custom uint64 to be unmarshallable
type Uint64 uint64

// UnmarshalJSON custom unmarshal function for json
func (u *Uint64) UnmarshalJSON(bs []byte) error {
	str := string(bs) // Parse plain numbers directly.
	if bs[0] == '"' && bs[len(bs)-1] == '"' {
		// Unwrap the quotes from string numbers.
		str = string(bs[1 : len(bs)-1])
	}
	x, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		return err
	}
	*u = Uint64(x)
	return nil
}

// UnmarshalYAML custom unmarshal function for yaml
func (u *Uint64) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	err := unmarshal(&str)
	if err != nil {
		return err
	}
	x, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		return err
	}
	*u = Uint64(x)

	return nil
}

// ProposerSettings is a Prysm internal representation of the fee recipient config on the validator client.
// ProposerSettingsPayload maps to ProposerSettings on import through the CLI.
type ProposerSettings struct {
	ProposeConfig map[[fieldparams.BLSPubkeyLength]byte]*ProposerOption
	DefaultConfig *ProposerOption
}

// ToPayload converts struct to ProposerSettingsPayload
func (ps *ProposerSettings) ToPayload() *ProposerSettingsPayload {
	payload := &ProposerSettingsPayload{
		ProposerConfig: make(map[string]*ProposerOptionPayload),
	}
	for key, option := range ps.ProposeConfig {
		payload.ProposerConfig[hexutil.Encode(key[:])] = &ProposerOptionPayload{
			FeeRecipient:  option.FeeRecipientConfig.FeeRecipient.Hex(),
			BuilderConfig: option.BuilderConfig.Clone(),
		}
	}
	if ps.DefaultConfig != nil {
		payload.DefaultConfig = &ProposerOptionPayload{
			FeeRecipient:  ps.DefaultConfig.FeeRecipientConfig.FeeRecipient.Hex(),
			BuilderConfig: ps.DefaultConfig.BuilderConfig.Clone(),
		}
	}
	return payload
}

// FeeRecipientConfig is a prysm internal representation to see if the fee recipient was set.
type FeeRecipientConfig struct {
	FeeRecipient common.Address
}

// ProposerOption is a Prysm internal representation of the ProposerOptionPayload on the validator client in bytes format instead of hex.
type ProposerOption struct {
	FeeRecipientConfig *FeeRecipientConfig
	BuilderConfig      *BuilderConfig
}

// Clone creates a deep copy of the proposer settings
func (ps *ProposerSettings) Clone() *ProposerSettings {
	clone := &ProposerSettings{
		ProposeConfig: make(map[[fieldparams.BLSPubkeyLength]byte]*ProposerOption),
		DefaultConfig: ps.DefaultConfig.Clone(),
	}
	for k, v := range ps.ProposeConfig {
		keyCopy := k
		valCopy := v.Clone()
		clone.ProposeConfig[keyCopy] = valCopy
	}
	return clone
}

// Clone creates a deep copy of fee recipient config
func (fo *FeeRecipientConfig) Clone() *FeeRecipientConfig {
	return &FeeRecipientConfig{fo.FeeRecipient}
}

// Clone creates a deep copy of builder config
func (bc *BuilderConfig) Clone() *BuilderConfig {
	relays := make([]string, len(bc.Relays))
	copy(relays, bc.Relays)
	return &BuilderConfig{bc.Enabled, bc.GasLimit, relays}
}

// Clone creates a deep copy of proposer option
func (po *ProposerOption) Clone() *ProposerOption {
	return &ProposerOption{
		FeeRecipientConfig: po.FeeRecipientConfig.Clone(),
		BuilderConfig:      po.BuilderConfig.Clone(),
	}
}
