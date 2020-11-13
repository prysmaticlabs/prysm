package slashingprotection

// Interchange defines the json scheme struct outlined in EIP3076.
// https://eips.ethereum.org/EIPS/eip-3076#json-schema
type Interchange struct {
	Metadata struct {
		InterchangeFormatVersion string `json:"interchange_format_version"`
		GenesisValidatorsRoot    string `json:"genesis_validators_root"`
	} `json:"metadata"`
	Data []struct {
		Pubkey       string `json:"pubkey"`
		SignedBlocks []struct {
			Properties struct {
				Slot        string `json:"slot"`
				SigningRoot string `json:"signing_root"`
			} `json:"properties"`
		} `json:"signed_blocks"`
		SignedAttestations []struct {
			Properties struct {
				SourceEpoch string `json:"source_epoch"`
				TargetEpoch string `json:"target_epoch"`
				SigningRoot string `json:"signing_root"`
			} `json:"properties"`
		} `json:"signed_attestations"`
	} `json:"data"`
}
