package slashingprotection

// Interchange defines the json scheme struct outlined in EIP3076.
// https://eips.ethereum.org/EIPS/eip-3076#json-schema
type Interchange struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Properties  struct {
		Metadata struct {
			Type       string `json:"type"`
			Properties struct {
				InterchangeFormatVersion struct {
					Type        string `json:"type"`
					Description string `json:"description"`
				} `json:"interchange_format_version"`
				GenesisValidatorsRoot struct {
					Type        string `json:"type"`
					Description string `json:"description"`
				} `json:"genesis_validators_root"`
			} `json:"properties"`
			Required []string `json:"required"`
		} `json:"metadata"`
		Data struct {
			Type  string `json:"type"`
			Items []struct {
				Type       string `json:"type"`
				Properties struct {
					Pubkey struct {
						Type        string `json:"type"`
						Description string `json:"description"`
					} `json:"pubkey"`
					SignedBlocks struct {
						Type  string `json:"type"`
						Items []struct {
							Type       string `json:"type"`
							Properties struct {
								Slot struct {
									Type        string `json:"type"`
									Description string `json:"description"`
								} `json:"slot"`
								SigningRoot struct {
									Type        string `json:"type"`
									Description string `json:"description"`
								} `json:"signing_root"`
							} `json:"properties"`
							Required []string `json:"required"`
						} `json:"items"`
					} `json:"signed_blocks"`
					SignedAttestations struct {
						Type  string `json:"type"`
						Items []struct {
							Type       string `json:"type"`
							Properties struct {
								SourceEpoch struct {
									Type        string `json:"type"`
									Description string `json:"description"`
								} `json:"source_epoch"`
								TargetEpoch struct {
									Type        string `json:"type"`
									Description string `json:"description"`
								} `json:"target_epoch"`
								SigningRoot struct {
									Type        string `json:"type"`
									Description string `json:"description"`
								} `json:"signing_root"`
							} `json:"properties"`
							Required []string `json:"required"`
						} `json:"items"`
					} `json:"signed_attestations"`
				} `json:"properties"`
				Required []string `json:"required"`
			} `json:"items"`
		} `json:"data"`
	} `json:"properties"`
	Required []string `json:"required"`
}
