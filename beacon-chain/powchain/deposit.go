package powchain

import (
	"fmt"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/stateutils"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

// processDeposit is a copy of the core function of the same name which includes some optimizations
// and removes the requirement to pass in beacon state. This is for determining genesis validators.
func (s *Service) processDepositOld(
	eth1Data *ethpb.Eth1Data,
	deposit *ethpb.Deposit,
) error {
	if err := verifyDeposit(eth1Data, deposit); err != nil {
		return errors.Wrapf(err, "could not verify deposit from %#x", bytesutil.Trunc(deposit.Data.PublicKey))
	}
	pubKey := bytesutil.ToBytes48(deposit.Data.PublicKey)
	amount := deposit.Data.Amount
	currBal, ok := s.depositedPubkeys[pubKey]
	if !ok {
		pub, err := bls.PublicKeyFromBytes(pubKey[:])
		if err != nil {
			return errors.Wrap(err, "could not deserialize validator public key")
		}
		domain := bls.ComputeDomain(params.BeaconConfig().DomainDeposit)
		sig, err := bls.SignatureFromBytes(deposit.Data.Signature)
		if err != nil {
			return errors.Wrap(err, "could not convert bytes to signature")
		}
		root, err := ssz.SigningRoot(deposit.Data)
		if err != nil {
			return errors.Wrap(err, "could not sign root for deposit data")
		}
		if !sig.Verify(root[:], pub, domain) {
			return fmt.Errorf("deposit signature did not verify")
		}
		s.depositedPubkeys[pubKey] = amount

		if amount >= params.BeaconConfig().MaxEffectiveBalance {
			s.activeValidatorCount++
		}
	} else {
		newBal := currBal + amount
		s.depositedPubkeys[pubKey] = newBal
		// exit if the validator is already an active validator previously
		if currBal >= params.BeaconConfig().MaxEffectiveBalance {
			return nil
		}
		if newBal >= params.BeaconConfig().MaxEffectiveBalance {
			s.activeValidatorCount++
		}
	}
	return nil
}

func verifyDeposit(eth1Data *ethpb.Eth1Data, deposit *ethpb.Deposit) error {
	// Verify Merkle proof of deposit and deposit trie root.
	receiptRoot := eth1Data.DepositRoot
	leaf, err := ssz.HashTreeRoot(deposit.Data)
	if err != nil {
		return errors.Wrap(err, "could not tree hash deposit data")
	}
	if ok := trieutil.VerifyMerkleProof(
		receiptRoot,
		leaf[:],
		int(eth1Data.DepositCount-1),
		deposit.Proof,
	); !ok {
		return fmt.Errorf(
			"deposit merkle branch of deposit root did not verify for root: %#x",
			receiptRoot,
		)
	}
	return nil
}

func (s *Service) processDeposit(eth1Data *ethpb.Eth1Data, deposit *ethpb.Deposit) error {
	var err error
	valIndexMap := stateutils.ValidatorIndexMap(s.preGenesisState)
	s.preGenesisState.Eth1Data = eth1Data
	s.preGenesisState, err = blocks.ProcessDeposit(s.preGenesisState, deposit, valIndexMap)
	if err != nil {
		return errors.Wrap(err, "could not process deposit")
	}
	pubkey := deposit.Data.PublicKey
	index, ok := valIndexMap[bytesutil.ToBytes48(pubkey)]
	if !ok {
		return nil
	}
	balance := s.preGenesisState.Balances[index]
	s.preGenesisState.Validators[index].EffectiveBalance = mathutil.Min(balance-balance%params.BeaconConfig().EffectiveBalanceIncrement, params.BeaconConfig().MaxEffectiveBalance)
	if s.preGenesisState.Validators[index].EffectiveBalance ==
		params.BeaconConfig().MaxEffectiveBalance {
		log.Errorf("adding in val %d", index)
		s.preGenesisState.Validators[index].ActivationEligibilityEpoch = 0
		s.preGenesisState.Validators[index].ActivationEpoch = 0
	}
	return nil
}
