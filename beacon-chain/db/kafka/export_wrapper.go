package kafka

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/iface"
	ethereum_beacon_p2p_v1 "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	eth "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/ethereum/go-ethereum/common"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

var _ = iface.Database(&Exporter{})

type Exporter struct {
	db iface.Database
	p *kafka.Producer
}

func Wrap(db iface.Database) (iface.Database, error) {
	p, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": "localhost"})
	if err != nil {
		return nil, err
	}

	return &Exporter{db:db, p:p}, nil
}

func (e Exporter) publish(ctx context.Context, topic string, msg proto.Message) {
	// TODO
}

func (e Exporter) Close() error {
	e.p.Close()
	return e.db.Close()
}

func (e Exporter) SaveAttestation(ctx context.Context, att *eth.Attestation) error {
	panic("implement me")
}

func (e Exporter) SaveAttestations(ctx context.Context, atts []*eth.Attestation) error {
	panic("implement me")
}

func (e Exporter) SaveBlock(ctx context.Context, block *eth.BeaconBlock) error {
	panic("implement me")
}

func (e Exporter) SaveBlocks(ctx context.Context, blocks []*eth.BeaconBlock) error {
	panic("implement me")
}

func (e Exporter) SaveHeadBlockRoot(ctx context.Context, blockRoot [32]byte) error {
	panic("implement me")
}

func (e Exporter) SaveGenesisBlockRoot(ctx context.Context, blockRoot [32]byte) error {
	panic("implement me")
}

func (e Exporter) SaveValidatorLatestVote(ctx context.Context, validatorIdx uint64, vote *ethereum_beacon_p2p_v1.ValidatorLatestVote) error {
	panic("implement me")
}

func (e Exporter) SaveValidatorLatestVotes(ctx context.Context, validatorIndices []uint64, votes []*ethereum_beacon_p2p_v1.ValidatorLatestVote) error {
	panic("implement me")
}

func (e Exporter) SaveValidatorIndex(ctx context.Context, publicKey [48]byte, validatorIdx uint64) error {
	panic("implement me")
}

func (e Exporter) SaveState(ctx context.Context, state *ethereum_beacon_p2p_v1.BeaconState, blockRoot [32]byte) error {
	panic("implement me")
}

func (e Exporter) SaveProposerSlashing(ctx context.Context, slashing *eth.ProposerSlashing) error {
	panic("implement me")
}

func (e Exporter) SaveAttesterSlashing(ctx context.Context, slashing *eth.AttesterSlashing) error {
	panic("implement me")
}

func (e Exporter) SaveVoluntaryExit(ctx context.Context, exit *eth.VoluntaryExit) error {
	panic("implement me")
}

func (e Exporter) SaveJustifiedCheckpoint(ctx context.Context, checkpoint *eth.Checkpoint) error {
	panic("implement me")
}

func (e Exporter) SaveFinalizedCheckpoint(ctx context.Context, checkpoint *eth.Checkpoint) error {
	panic("implement me")
}

func (e Exporter) SaveArchivedActiveValidatorChanges(ctx context.Context, epoch uint64, changes *eth.ArchivedActiveSetChanges) error {
	panic("implement me")
}

func (e Exporter) SaveArchivedCommitteeInfo(ctx context.Context, epoch uint64, info *eth.ArchivedCommitteeInfo) error {
	panic("implement me")
}

func (e Exporter) SaveArchivedBalances(ctx context.Context, epoch uint64, balances []uint64) error {
	panic("implement me")
}

func (e Exporter) SaveArchivedValidatorParticipation(ctx context.Context, epoch uint64, part *eth.ValidatorParticipation) error {
	panic("implement me")
}

func (e Exporter) SaveDepositContractAddress(ctx context.Context, addr common.Address) error {
	panic("implement me")
}
