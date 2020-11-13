package interchangeformat

// EIPSlashingProtectionFormat string representation of a standard
// format for representing validator slashing protection db data.
type EIPSlashingProtectionFormat struct {
	Metadata struct {
		InterchangeFormatVersion string `json:"interchange_format_version"`
		GenesisValidatorsRoot    string `json:"genesis_validators_root"`
	} `json:"metadata"`
	Data []Data `json:"data"`
}

// Data field for the standard slashing protection format.
type Data struct {
	Pubkey             string               `json:"pubkey"`
	SignedBlocks       []*SignedBlock       `json:"signed_blocks"`
	SignedAttestations []*SignedAttestation `json:"signed_attestations"`
}

// SignedAttestation in the standard slashing protection format file, including
// a source epoch, target epoch, and an optional signing root.
type SignedAttestation struct {
	SourceEpoch string `json:"source_epoch"`
	TargetEpoch string `json:"target_epoch"`
	SigningRoot string `json:"signing_root,omitempty"`
}

// SignedBlock in the standard slashing protection format, including a slot
// and an optional signing root.
type SignedBlock struct {
	Slot        string `json:"slot"`
	SigningRoot string `json:"signing_root,omitempty"`
}
