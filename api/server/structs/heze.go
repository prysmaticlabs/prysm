package structs

import "encoding/json"

// ExecutionPayloadBidHeze is the Heze execution payload bid: the Gloas fields plus inclusion_list_bits.
type ExecutionPayloadBidHeze struct {
	ExecutionPayloadBid
	InclusionListBits string `json:"inclusion_list_bits"`
}

type SignedExecutionPayloadBidHeze struct {
	Message   *ExecutionPayloadBidHeze `json:"message"`
	Signature string                   `json:"signature"`
}

// BeaconBlockBodyHeze is the Gloas body with the Heze bid.
// TODO(heze): Eth1Data and Deposits stay until EIP-8015.
type BeaconBlockBodyHeze struct {
	RandaoReveal              string                         `json:"randao_reveal"`
	Eth1Data                  *Eth1Data                      `json:"eth1_data"`
	Graffiti                  string                         `json:"graffiti"`
	ProposerSlashings         []*ProposerSlashing            `json:"proposer_slashings"`
	AttesterSlashings         []*AttesterSlashingElectra     `json:"attester_slashings"`
	Attestations              []*AttestationElectra          `json:"attestations"`
	Deposits                  []*Deposit                     `json:"deposits"`
	VoluntaryExits            []*SignedVoluntaryExit         `json:"voluntary_exits"`
	SyncAggregate             *SyncAggregate                 `json:"sync_aggregate"`
	BLSToExecutionChanges     []*SignedBLSToExecutionChange  `json:"bls_to_execution_changes"`
	SignedExecutionPayloadBid *SignedExecutionPayloadBidHeze `json:"signed_execution_payload_bid"`
	PayloadAttestations       []*PayloadAttestation          `json:"payload_attestations"`
	ParentExecutionRequests   *ExecutionRequestsGloas        `json:"parent_execution_requests"`
}

type BeaconBlockHeze struct {
	Slot          string               `json:"slot"`
	ProposerIndex string               `json:"proposer_index"`
	ParentRoot    string               `json:"parent_root"`
	StateRoot     string               `json:"state_root"`
	Body          *BeaconBlockBodyHeze `json:"body"`
}

type SignedBeaconBlockHeze struct {
	Message   *BeaconBlockHeze `json:"message"`
	Signature string           `json:"signature"`
}

var _ SignedMessageJsoner = &SignedBeaconBlockHeze{}

func (s *SignedBeaconBlockHeze) MessageRawJson() ([]byte, error) {
	return json.Marshal(s.Message)
}

func (s *SignedBeaconBlockHeze) SigString() string {
	return s.Signature
}
