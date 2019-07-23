package powchain

import (
	"fmt"

	"github.com/prysmaticlabs/go-ssz"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

// processDeposit is a copy of the core function of the same name which includes some optimizations
// and removes the requirement to pass in beacon state. This is for determining genesis validators.
func (w *Web3Service) processDeposit(
	eth1Data *ethpb.Eth1Data,
	deposit *ethpb.Deposit,
) error {
	if err := verifyDeposit(eth1Data, deposit); err != nil {
		return fmt.Errorf("could not verify deposit from #%x: %v", bytesutil.Trunc(deposit.Data.PublicKey), err)
	}
	pubKey := bytesutil.ToBytes48(deposit.Data.PublicKey)
	amount := deposit.Data.Amount
	currBal, ok := w.depositedPubkeys[pubKey]
	if !ok {
		pub, err := bls.PublicKeyFromBytes(pubKey[:])
		if err != nil {
			return fmt.Errorf("could not deserialize validator public key: %v", err)
		}
		domain := bls.Domain(params.BeaconConfig().DomainDeposit, params.BeaconConfig().GenesisForkVersion)
		sig, err := bls.SignatureFromBytes(deposit.Data.Signature)
		if err != nil {
			return fmt.Errorf("could not convert bytes to signature: %v", err)
		}
		root, err := ssz.SigningRoot(deposit.Data)
		if err != nil {
			return fmt.Errorf("could not sign root for deposit data: %v", err)
		}
		if !sig.Verify(root[:], pub, domain) {
			return fmt.Errorf("deposit signature did not verify")
		}
		w.depositedPubkeys[pubKey] = amount

		if amount >= params.BeaconConfig().MaxEffectiveBalance {
			w.activeValidatorCount++
		}
	} else {
		newBal := currBal + amount
		w.depositedPubkeys[pubKey] = newBal
		// exit if the validator is already an active validator previously
		if currBal >= params.BeaconConfig().MaxEffectiveBalance {
			return nil
		}
		if newBal >= params.BeaconConfig().MaxEffectiveBalance {
			w.activeValidatorCount++
		}
	}
	return nil
}

func verifyDeposit(eth1Data *ethpb.Eth1Data, deposit *ethpb.Deposit) error {
	// Verify Merkle proof of deposit and deposit trie root.
	receiptRoot := eth1Data.DepositRoot
	leaf, err := hashutil.DepositHash(deposit.Data)
	if err != nil {
		return fmt.Errorf("could not tree hash deposit data: %v", err)
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
