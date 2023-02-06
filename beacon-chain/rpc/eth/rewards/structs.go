package rewards

import (
	"encoding/json"
	"strconv"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
)

type BlockRewardsRequest struct {
	BlockId string `json:"block_id"`
}

type BlockRewardsResponse struct {
	Data                *BlockRewards `json:"data"`
	ExecutionOptimistic bool          `json:"execution_optimistic"`
	Finalized           bool          `json:"finalized"`
}

type BlockRewards struct {
	ProposerIndex     types.ValidatorIndex
	Total             uint64
	Attestations      uint64
	SyncAggregate     uint64
	ProposerSlashings uint64
	AttesterSlashings uint64
}

type BlockRewardsJson struct {
	ProposerIndex     string `json:"proposer_index"`
	Total             string `json:"total"`
	Attestations      string `json:"attestations"`
	SyncAggregate     string `json:"sync_aggregate"`
	ProposerSlashings string `json:"proposer_slashings"`
	AttesterSlashings string `json:"attester_slashings"`
}

func (r *BlockRewards) MarshalJSON() ([]byte, error) {
	return json.Marshal(&BlockRewardsJson{
		ProposerIndex:     strconv.FormatUint(uint64(r.ProposerIndex), 10),
		Total:             strconv.FormatUint(r.Total, 10),
		Attestations:      strconv.FormatUint(r.Attestations, 10),
		SyncAggregate:     strconv.FormatUint(r.SyncAggregate, 10),
		ProposerSlashings: strconv.FormatUint(r.ProposerSlashings, 10),
		AttesterSlashings: strconv.FormatUint(r.AttesterSlashings, 10),
	})
}

func (r *BlockRewards) UnmarshalJSON(b []byte) error {
	j := &BlockRewardsJson{}
	err := json.Unmarshal(b, j)
	if err != nil {
		return err
	}
	proposerIndex, err := strconv.ParseUint(j.ProposerIndex, 10, 64)
	if err != nil {
		return errors.Wrapf(err, "could not unmarshal proposer index")
	}
	total, err := strconv.ParseUint(j.Total, 10, 64)
	if err != nil {
		return errors.Wrapf(err, "could not unmarshal total")
	}
	attestations, err := strconv.ParseUint(j.Attestations, 10, 64)
	if err != nil {
		return errors.Wrapf(err, "could not unmarshal attestations")
	}
	syncAggregate, err := strconv.ParseUint(j.SyncAggregate, 10, 64)
	if err != nil {
		return errors.Wrapf(err, "could not unmarshal sync aggregate")
	}
	proposerSlashings, err := strconv.ParseUint(j.ProposerSlashings, 10, 64)
	if err != nil {
		return errors.Wrapf(err, "could not unmarshal proposer slashings")
	}
	attesterSlashings, err := strconv.ParseUint(j.AttesterSlashings, 10, 64)
	if err != nil {
		return errors.Wrapf(err, "could not unmarshal attester slashings")
	}
	*r = BlockRewards{
		ProposerIndex:     types.ValidatorIndex(proposerIndex),
		Total:             total,
		Attestations:      attestations,
		SyncAggregate:     syncAggregate,
		ProposerSlashings: proposerSlashings,
		AttesterSlashings: attesterSlashings,
	}
	return nil
}
