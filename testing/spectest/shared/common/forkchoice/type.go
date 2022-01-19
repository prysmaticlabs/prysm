package forkchoice

type Step struct {
	Tick        *int    `json:"tick"`
	Block       *string `json:"block"`
	Valid       *bool   `json:"valid"`
	Attestation *string `json:"attestation"`
	Check       *Check  `json:"checks"`
}

type Check struct {
	Time                    *int       `json:"time"`
	GenesisTime             int        `json:"genesis_time"`
	ProposerBoostRoot       *string     `json:"proposer_boost_root"`
	Head                    *SlotRoot  `json:"head"`
	JustifiedCheckPoint     *EpochRoot `json:"justified_checkpoint"`
	BestJustifiedCheckPoint *EpochRoot `json:"best_justified_checkpoint"`
	FinalizedCheckPoint     *EpochRoot `json:"finalized_checkpoint"`
}

type SlotRoot struct {
	Slot int    `json:"slot"`
	Root string `json:"root"`
}

type EpochRoot struct {
	Epoch int    `json:"epoch"`
	Root  string `json:"root"`
}
