package beaconapi_evaluators

type validatorJson struct {
	PublicKey                  string `json:"pubkey"`
	WithdrawalCredentials      string `json:"withdrawal_credentials"`
	EffectiveBalance           string `json:"effective_balance"`
	Slashed                    bool   `json:"slashed"`
	ActivationEligibilityEpoch string `json:"activation_eligibility_epoch"`
	ActivationEpoch            string `json:"activation_epoch"`
	ExitEpoch                  string `json:"exit_epoch"`
	WithdrawableEpoch          string `json:"withdrawable_epoch"`
}

type validatorContainerJson struct {
	Index     string         `json:"index"`
	Balance   string         `json:"balance"`
	Status    string         `json:"status"`
	Validator *validatorJson `json:"validator"`
}

type blockResponseJson struct {
	Data *signedBeaconBlockContainerJson `json:"data"`
}

type signedBeaconBlockContainerJson struct {
	Message   *beaconBlockJson `json:"message"`
	Signature string           `json:"signature" hex:"true"`
}

type beaconBlockJson struct {
	Slot          string               `json:"slot"`
	ProposerIndex string               `json:"proposer_index"`
	ParentRoot    string               `json:"parent_root" hex:"true"`
	StateRoot     string               `json:"state_root" hex:"true"`
	Body          *beaconBlockBodyJson `json:"body"`
}

type beaconBlockBodyJson struct {
	RandaoReveal      string                     `json:"randao_reveal" hex:"true"`
	Eth1Data          *eth1DataJson              `json:"eth1_data"`
	Graffiti          string                     `json:"graffiti" hex:"true"`
	ProposerSlashings []*proposerSlashingJson    `json:"proposer_slashings"`
	AttesterSlashings []*attesterSlashingJson    `json:"attester_slashings"`
	Attestations      []*attestationJson         `json:"attestations"`
	Deposits          []*depositJson             `json:"deposits"`
	VoluntaryExits    []*signedVoluntaryExitJson `json:"voluntary_exits"`
}

type eth1DataJson struct {
	DepositRoot  string `json:"deposit_root" hex:"true"`
	DepositCount string `json:"deposit_count"`
	BlockHash    string `json:"block_hash" hex:"true"`
}

type depositJson struct {
	Proof []string          `json:"proof" hex:"true"`
	Data  *deposit_DataJson `json:"data"`
}

type deposit_DataJson struct {
	PublicKey             string `json:"pubkey" hex:"true"`
	WithdrawalCredentials string `json:"withdrawal_credentials" hex:"true"`
	Amount                string `json:"amount"`
	Signature             string `json:"signature" hex:"true"`
}

type proposerSlashingJson struct {
	Header_1 *signedBeaconBlockHeaderJson `json:"signed_header_1"`
	Header_2 *signedBeaconBlockHeaderJson `json:"signed_header_2"`
}
type signedBeaconBlockHeaderJson struct {
	Header    *beaconBlockHeaderJson `json:"message"`
	Signature string                 `json:"signature" hex:"true"`
}
type beaconBlockHeaderJson struct {
	Slot          string `json:"slot"`
	ProposerIndex string `json:"proposer_index"`
	ParentRoot    string `json:"parent_root" hex:"true"`
	StateRoot     string `json:"state_root" hex:"true"`
	BodyRoot      string `json:"body_root" hex:"true"`
}

type attesterSlashingJson struct {
	Attestation_1 *indexedAttestationJson `json:"attestation_1"`
	Attestation_2 *indexedAttestationJson `json:"attestation_2"`
}

type indexedAttestationJson struct {
	AttestingIndices []string             `json:"attesting_indices"`
	Data             *attestationDataJson `json:"data"`
	Signature        string               `json:"signature" hex:"true"`
}

type attestationJson struct {
	AggregationBits string               `json:"aggregation_bits" hex:"true"`
	Data            *attestationDataJson `json:"data"`
	Signature       string               `json:"signature" hex:"true"`
}

type attestationDataJson struct {
	Slot            string          `json:"slot"`
	CommitteeIndex  string          `json:"index"`
	BeaconBlockRoot string          `json:"beacon_block_root" hex:"true"`
	Source          *checkpointJson `json:"source"`
	Target          *checkpointJson `json:"target"`
}

type signedVoluntaryExitJson struct {
	Exit      *voluntaryExitJson `json:"message"`
	Signature string             `json:"signature" hex:"true"`
}

type voluntaryExitJson struct {
	Epoch          string `json:"epoch"`
	ValidatorIndex string `json:"validator_index"`
}
type checkpointJson struct {
	Epoch string `json:"epoch"`
	Root  string `json:"root" hex:"true"`
}
