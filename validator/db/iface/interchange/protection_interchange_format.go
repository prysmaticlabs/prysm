package interchange

// PlainDataInterchangeFormat string representation of the data interchange
// format of validator slashing protection db data.
type PlainDataInterchangeFormat struct {
	Metadata struct {
		InterchangeFormatVersion string `json:"interchange_format_version"`
		GenesisValidatorsRoot    string `json:"genesis_validators_root"`
	} `json:"metadata"`
	Data []Data `json:"data"`
}
type Data struct {
	Pubkey             string               `json:"pubkey"`
	SignedBlocks       []SignedBlocks       `json:"signed_blocks"`
	SignedAttestations []SignedAttestations `json:"signed_attestations"`
}
type SignedAttestations struct {
	SourceEpoch string `json:"source_epoch"`
	TargetEpoch string `json:"target_epoch"`
	SigningRoot string `json:"signing_root,omitempty"`
}
type SignedBlocks struct {
	Slot        string `json:"slot"`
	SigningRoot string `json:"signing_root,omitempty"`
}

// DataInterchangeFormat represents the data interchange format of validator slashing protection db data.
type DataInterchangeFormat struct {
	Metadata struct {
		InterchangeFormatVersion uint64
		GenesisValidatorsRoot    [32]byte
	}
	Data []struct {
		Pubkey       [48]byte
		SignedBlocks []struct {
			Slot        uint64
			SigningRoot [32]byte
		}
		SignedAttestations []struct {
			SourceEpoch uint64
			TargetEpoch uint64
			SigningRoot [32]byte
		}
	}
}
