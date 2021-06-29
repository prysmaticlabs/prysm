package state

import (
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

// CopyETH1Data copies the provided eth1data object.
func CopyETH1Data(data *ethpb.Eth1Data) *ethpb.Eth1Data {
	if data == nil {
		return nil
	}
	return &ethpb.Eth1Data{
		DepositRoot:  bytesutil.SafeCopyBytes(data.DepositRoot),
		DepositCount: data.DepositCount,
		BlockHash:    bytesutil.SafeCopyBytes(data.BlockHash),
	}
}

// CopyPendingAttestation copies the provided pending attestation object.
func CopyPendingAttestation(att *pbp2p.PendingAttestation) *pbp2p.PendingAttestation {
	if att == nil {
		return nil
	}
	data := CopyAttestationData(att.Data)
	return &pbp2p.PendingAttestation{
		AggregationBits: bytesutil.SafeCopyBytes(att.AggregationBits),
		Data:            data,
		InclusionDelay:  att.InclusionDelay,
		ProposerIndex:   att.ProposerIndex,
	}
}

// CopyAttestation copies the provided attestation object.
func CopyAttestation(att *ethpb.Attestation) *ethpb.Attestation {
	if att == nil {
		return nil
	}
	return &ethpb.Attestation{
		AggregationBits: bytesutil.SafeCopyBytes(att.AggregationBits),
		Data:            CopyAttestationData(att.Data),
		Signature:       bytesutil.SafeCopyBytes(att.Signature),
	}
}

// CopyAttestationData copies the provided AttestationData object.
func CopyAttestationData(attData *ethpb.AttestationData) *ethpb.AttestationData {
	if attData == nil {
		return nil
	}
	return &ethpb.AttestationData{
		Slot:            attData.Slot,
		CommitteeIndex:  attData.CommitteeIndex,
		BeaconBlockRoot: bytesutil.SafeCopyBytes(attData.BeaconBlockRoot),
		Source:          CopyCheckpoint(attData.Source),
		Target:          CopyCheckpoint(attData.Target),
	}
}

// CopyCheckpoint copies the provided checkpoint.
func CopyCheckpoint(cp *ethpb.Checkpoint) *ethpb.Checkpoint {
	if cp == nil {
		return nil
	}
	return &ethpb.Checkpoint{
		Epoch: cp.Epoch,
		Root:  bytesutil.SafeCopyBytes(cp.Root),
	}
}

// CopySignedBeaconBlock copies the provided SignedBeaconBlock.
func CopySignedBeaconBlock(sigBlock *ethpb.SignedBeaconBlock) *ethpb.SignedBeaconBlock {
	if sigBlock == nil {
		return nil
	}
	return &ethpb.SignedBeaconBlock{
		Block:     CopyBeaconBlock(sigBlock.Block),
		Signature: bytesutil.SafeCopyBytes(sigBlock.Signature),
	}
}

// CopyBeaconBlock copies the provided BeaconBlock.
func CopyBeaconBlock(block *ethpb.BeaconBlock) *ethpb.BeaconBlock {
	if block == nil {
		return nil
	}
	return &ethpb.BeaconBlock{
		Slot:          block.Slot,
		ProposerIndex: block.ProposerIndex,
		ParentRoot:    bytesutil.SafeCopyBytes(block.ParentRoot),
		StateRoot:     bytesutil.SafeCopyBytes(block.StateRoot),
		Body:          CopyBeaconBlockBody(block.Body),
	}
}

// CopyBeaconBlockBody copies the provided BeaconBlockBody.
func CopyBeaconBlockBody(body *ethpb.BeaconBlockBody) *ethpb.BeaconBlockBody {
	if body == nil {
		return nil
	}
	return &ethpb.BeaconBlockBody{
		RandaoReveal:      bytesutil.SafeCopyBytes(body.RandaoReveal),
		Eth1Data:          CopyETH1Data(body.Eth1Data),
		Graffiti:          bytesutil.SafeCopyBytes(body.Graffiti),
		ProposerSlashings: CopyProposerSlashings(body.ProposerSlashings),
		AttesterSlashings: CopyAttesterSlashings(body.AttesterSlashings),
		Attestations:      CopyAttestations(body.Attestations),
		Deposits:          CopyDeposits(body.Deposits),
		VoluntaryExits:    CopySignedVoluntaryExits(body.VoluntaryExits),
		PandoraShard:      CopyPandoraShard(body.PandoraShard),
	}
}

// CopyProposerSlashings copies the provided ProposerSlashing array.
func CopyProposerSlashings(slashings []*ethpb.ProposerSlashing) []*ethpb.ProposerSlashing {
	if slashings == nil {
		return nil
	}
	newSlashings := make([]*ethpb.ProposerSlashing, len(slashings))
	for i, att := range slashings {
		newSlashings[i] = CopyProposerSlashing(att)
	}
	return newSlashings
}

// CopyProposerSlashing copies the provided ProposerSlashing.
func CopyProposerSlashing(slashing *ethpb.ProposerSlashing) *ethpb.ProposerSlashing {
	if slashing == nil {
		return nil
	}
	return &ethpb.ProposerSlashing{
		Header_1: CopySignedBeaconBlockHeader(slashing.Header_1),
		Header_2: CopySignedBeaconBlockHeader(slashing.Header_2),
	}
}

// CopySignedBeaconBlockHeader copies the provided SignedBeaconBlockHeader.
func CopySignedBeaconBlockHeader(header *ethpb.SignedBeaconBlockHeader) *ethpb.SignedBeaconBlockHeader {
	if header == nil {
		return nil
	}
	return &ethpb.SignedBeaconBlockHeader{
		Header:    CopyBeaconBlockHeader(header.Header),
		Signature: bytesutil.SafeCopyBytes(header.Signature),
	}
}

// CopyBeaconBlockHeader copies the provided BeaconBlockHeader.
func CopyBeaconBlockHeader(header *ethpb.BeaconBlockHeader) *ethpb.BeaconBlockHeader {
	if header == nil {
		return nil
	}
	parentRoot := bytesutil.SafeCopyBytes(header.ParentRoot)
	stateRoot := bytesutil.SafeCopyBytes(header.StateRoot)
	bodyRoot := bytesutil.SafeCopyBytes(header.BodyRoot)
	return &ethpb.BeaconBlockHeader{
		Slot:          header.Slot,
		ProposerIndex: header.ProposerIndex,
		ParentRoot:    parentRoot,
		StateRoot:     stateRoot,
		BodyRoot:      bodyRoot,
	}
}

// CopyAttesterSlashings copies the provided AttesterSlashings array.
func CopyAttesterSlashings(slashings []*ethpb.AttesterSlashing) []*ethpb.AttesterSlashing {
	if slashings == nil {
		return nil
	}
	newSlashings := make([]*ethpb.AttesterSlashing, len(slashings))
	for i, slashing := range slashings {
		newSlashings[i] = &ethpb.AttesterSlashing{
			Attestation_1: CopyIndexedAttestation(slashing.Attestation_1),
			Attestation_2: CopyIndexedAttestation(slashing.Attestation_2),
		}
	}
	return newSlashings
}

// CopyIndexedAttestation copies the provided IndexedAttestation.
func CopyIndexedAttestation(indexedAtt *ethpb.IndexedAttestation) *ethpb.IndexedAttestation {
	var indices []uint64
	if indexedAtt == nil {
		return nil
	} else if indexedAtt.AttestingIndices != nil {
		indices = make([]uint64, len(indexedAtt.AttestingIndices))
		copy(indices, indexedAtt.AttestingIndices)
	}
	return &ethpb.IndexedAttestation{
		AttestingIndices: indices,
		Data:             CopyAttestationData(indexedAtt.Data),
		Signature:        bytesutil.SafeCopyBytes(indexedAtt.Signature),
	}
}

// CopyAttestations copies the provided Attestation array.
func CopyAttestations(attestations []*ethpb.Attestation) []*ethpb.Attestation {
	if attestations == nil {
		return nil
	}
	newAttestations := make([]*ethpb.Attestation, len(attestations))
	for i, att := range attestations {
		newAttestations[i] = CopyAttestation(att)
	}
	return newAttestations
}

// CopyDeposits copies the provided deposit array.
func CopyDeposits(deposits []*ethpb.Deposit) []*ethpb.Deposit {
	if deposits == nil {
		return nil
	}
	newDeposits := make([]*ethpb.Deposit, len(deposits))
	for i, dep := range deposits {
		newDeposits[i] = CopyDeposit(dep)
	}
	return newDeposits
}

// CopyDeposit copies the provided deposit.
func CopyDeposit(deposit *ethpb.Deposit) *ethpb.Deposit {
	if deposit == nil {
		return nil
	}
	return &ethpb.Deposit{
		Proof: bytesutil.Copy2dBytes(deposit.Proof),
		Data:  CopyDepositData(deposit.Data),
	}
}

// CopyDepositData copies the provided deposit data.
func CopyDepositData(depData *ethpb.Deposit_Data) *ethpb.Deposit_Data {
	if depData == nil {
		return nil
	}
	return &ethpb.Deposit_Data{
		PublicKey:             bytesutil.SafeCopyBytes(depData.PublicKey),
		WithdrawalCredentials: bytesutil.SafeCopyBytes(depData.WithdrawalCredentials),
		Amount:                depData.Amount,
		Signature:             bytesutil.SafeCopyBytes(depData.Signature),
	}
}

// CopySignedVoluntaryExits copies the provided SignedVoluntaryExits array.
func CopySignedVoluntaryExits(exits []*ethpb.SignedVoluntaryExit) []*ethpb.SignedVoluntaryExit {
	if exits == nil {
		return nil
	}
	newExits := make([]*ethpb.SignedVoluntaryExit, len(exits))
	for i, exit := range exits {
		newExits[i] = CopySignedVoluntaryExit(exit)
	}
	return newExits
}

// CopySignedVoluntaryExit copies the provided SignedVoluntaryExit.
func CopySignedVoluntaryExit(exit *ethpb.SignedVoluntaryExit) *ethpb.SignedVoluntaryExit {
	if exit == nil {
		return nil
	}
	return &ethpb.SignedVoluntaryExit{
		Exit: &ethpb.VoluntaryExit{
			Epoch:          exit.Exit.Epoch,
			ValidatorIndex: exit.Exit.ValidatorIndex,
		},
		Signature: bytesutil.SafeCopyBytes(exit.Signature),
	}
}

// CopyValidator copies the provided validator.
func CopyValidator(val *ethpb.Validator) *ethpb.Validator {
	pubKey := make([]byte, len(val.PublicKey))
	copy(pubKey, val.PublicKey)
	withdrawalCreds := make([]byte, len(val.WithdrawalCredentials))
	copy(withdrawalCreds, val.WithdrawalCredentials)
	return &ethpb.Validator{
		PublicKey:                  pubKey,
		WithdrawalCredentials:      withdrawalCreds,
		EffectiveBalance:           val.EffectiveBalance,
		Slashed:                    val.Slashed,
		ActivationEligibilityEpoch: val.ActivationEligibilityEpoch,
		ActivationEpoch:            val.ActivationEpoch,
		ExitEpoch:                  val.ExitEpoch,
		WithdrawableEpoch:          val.WithdrawableEpoch,
	}
}

// CopyPandoraShard copies PandoraShard
func CopyPandoraShard(pShards []*ethpb.PandoraShard) []*ethpb.PandoraShard {
	if len(pShards) == 0 {
		return nil
	}
	pandoraShards := make([]*ethpb.PandoraShard, 1)
	ps := pShards[0]
	pandoraShard := &ethpb.PandoraShard{
		BlockNumber: ps.BlockNumber,
		Hash:        ps.Hash,
		ParentHash:  ps.ParentHash,
		StateRoot:   ps.StateRoot,
		TxHash:      ps.TxHash,
		ReceiptHash: ps.ReceiptHash,
		SealHash:    ps.SealHash,
		Signature:   ps.Signature,
	}
	pandoraShards[0] = pandoraShard
	return pandoraShards
}
