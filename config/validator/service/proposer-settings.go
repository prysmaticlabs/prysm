package validator_service_config

import (
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
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
	settings := &ProposerSettings{}
	if ps.ProposerConfig != nil {
		settings.ProposeConfig = make(map[[fieldparams.BLSPubkeyLength]byte]*ProposerOption)
		for key, optionPayload := range ps.ProposerConfig {
			if optionPayload.FeeRecipient == "" {
				continue
			}
			b, err := hexutil.Decode(key)
			if err != nil {
				return nil, err
			}
			p := &ProposerOption{
				FeeRecipientConfig: &FeeRecipientConfig{
					FeeRecipient: common.HexToAddress(optionPayload.FeeRecipient),
				},
			}
			if optionPayload.BuilderConfig != nil {
				p.BuilderConfig = optionPayload.BuilderConfig.Clone()
			}
			settings.ProposeConfig[bytesutil.ToBytes48(b)] = p
		}
	}
	if ps.DefaultConfig != nil {
		d := &ProposerOption{}
		if ps.DefaultConfig.FeeRecipient != "" {
			d.FeeRecipientConfig = &FeeRecipientConfig{
				FeeRecipient: common.HexToAddress(ps.DefaultConfig.FeeRecipient),
			}
		}
		if ps.DefaultConfig.BuilderConfig != nil {
			d.BuilderConfig = ps.DefaultConfig.BuilderConfig.Clone()
		}
		settings.DefaultConfig = d
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
	if ps == nil {
		return nil
	}
	payload := &ProposerSettingsPayload{
		ProposerConfig: make(map[string]*ProposerOptionPayload),
	}
	for key, option := range ps.ProposeConfig {
		p := &ProposerOptionPayload{}
		if option.FeeRecipientConfig != nil {
			p.FeeRecipient = option.FeeRecipientConfig.FeeRecipient.Hex()
		}
		if option.BuilderConfig != nil {
			p.BuilderConfig = option.BuilderConfig.Clone()
		}
		payload.ProposerConfig[hexutil.Encode(key[:])] = p
	}
	if ps.DefaultConfig != nil {
		p := &ProposerOptionPayload{}
		if ps.DefaultConfig.FeeRecipientConfig != nil {
			p.FeeRecipient = ps.DefaultConfig.FeeRecipientConfig.FeeRecipient.Hex()
		}
		if ps.DefaultConfig.BuilderConfig != nil {
			p.BuilderConfig = ps.DefaultConfig.BuilderConfig.Clone()
		}
		payload.DefaultConfig = p
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
	if ps == nil {
		return nil
	}
	clone := &ProposerSettings{}
	if ps.DefaultConfig != nil {
		clone.DefaultConfig = ps.DefaultConfig.Clone()
	}
	if ps.ProposeConfig != nil {
		clone.ProposeConfig = make(map[[fieldparams.BLSPubkeyLength]byte]*ProposerOption)
		for k, v := range ps.ProposeConfig {
			keyCopy := k
			valCopy := v.Clone()
			clone.ProposeConfig[keyCopy] = valCopy
		}
	}

	return clone
}

// Clone creates a deep copy of fee recipient config
func (fo *FeeRecipientConfig) Clone() *FeeRecipientConfig {
	if fo == nil {
		return nil
	}
	return &FeeRecipientConfig{fo.FeeRecipient}
}

// Clone creates a deep copy of builder config
func (bc *BuilderConfig) Clone() *BuilderConfig {
	if bc == nil {
		return nil
	}
	relays := make([]string, len(bc.Relays))
	copy(relays, bc.Relays)
	return &BuilderConfig{bc.Enabled, bc.GasLimit, relays}
}

// Clone creates a deep copy of proposer option
func (po *ProposerOption) Clone() *ProposerOption {
	if po == nil {
		return nil
	}
	p := &ProposerOption{}
	if po.FeeRecipientConfig != nil {
		p.FeeRecipientConfig = po.FeeRecipientConfig.Clone()
	}
	if po.BuilderConfig != nil {
		p.BuilderConfig = po.BuilderConfig.Clone()
	}
	return p
}
