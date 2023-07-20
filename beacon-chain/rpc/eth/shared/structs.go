package shared

type Attestation struct {
	AggregationBits string          `json:"aggregation_bits"`
	Data            AttestationData `json:"data"`
	Signature       string          `json:"signature"`
}

type AttestationData struct {
	Slot            string     `json:"slot"`
	CommitteeIndex  string     `json:"index"`
	BeaconBlockRoot string     `json:"beacon_block_root"`
	Source          Checkpoint `json:"source"`
	Target          Checkpoint `json:"target"`
}

type Checkpoint struct {
	Epoch string `json:"epoch"`
	Root  string `json:"root"`
}
