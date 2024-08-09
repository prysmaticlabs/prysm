package altair

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

// ProcessPreGenesisDeposits processes a deposit for the beacon state before chainstart.
func ProcessPreGenesisDeposits(
	ctx context.Context,
	beaconState state.BeaconState,
	deposits []*ethpb.Deposit,
) (state.BeaconState, error) {
	var err error
	beaconState, err = ProcessDeposits(ctx, beaconState, deposits)
	if err != nil {
		return nil, errors.Wrap(err, "could not process deposit")
	}
	beaconState, err = blocks.ActivateValidatorWithEffectiveBalance(beaconState, deposits)
	if err != nil {
		return nil, err
	}
	return beaconState, nil
}

// ProcessDeposits processes validator deposits for beacon state Altair.
func ProcessDeposits(
	ctx context.Context,
	beaconState state.BeaconState,
	deposits []*ethpb.Deposit,
) (state.BeaconState, error) {
	batchVerified, err := blocks.BatchVerifyDepositsSignatures(ctx, deposits)
	if err != nil {
		return nil, err
	}

	for _, deposit := range deposits {
		if deposit == nil || deposit.Data == nil {
			return nil, errors.New("got a nil deposit in block")
		}
		beaconState, err = ProcessDeposit(beaconState, deposit, batchVerified)
		if err != nil {
			return nil, errors.Wrapf(err, "could not process deposit from %#x", bytesutil.Trunc(deposit.Data.PublicKey))
		}
	}
	return beaconState, nil
}

// ProcessDeposit takes in a deposit object and inserts it
// into the registry as a new validator or balance change.
// Returns the resulting state, a boolean to indicate whether or not the deposit
// resulted in a new validator entry into the beacon state, and any error.
//
// Spec pseudocode definition:
// def process_deposit(state: BeaconState, deposit: Deposit) -> None:
//
//	# Verify the Merkle branch
//	assert is_valid_merkle_branch(
//		leaf=hash_tree_root(deposit.data),
//		branch=deposit.proof,
//		depth=DEPOSIT_CONTRACT_TREE_DEPTH + 1,  # Add 1 for the List length mix-in
//		index=state.eth1_deposit_index,
//		root=state.eth1_data.deposit_root,
//	)
//
//	 # Deposits must be processed in order
//	 state.eth1_deposit_index += 1
//
//	 apply_deposit(
//	  state=state,
//	  pubkey=deposit.data.pubkey,
//	  withdrawal_credentials=deposit.data.withdrawal_credentials,
//	  amount=deposit.data.amount,
//	  signature=deposit.data.signature,
//	 )
func ProcessDeposit(beaconState state.BeaconState, deposit *ethpb.Deposit, verifySignature bool) (state.BeaconState, error) {
	if err := blocks.VerifyDeposit(beaconState, deposit); err != nil {
		if deposit == nil || deposit.Data == nil {
			return nil, err
		}
		return nil, errors.Wrapf(err, "could not verify deposit from %#x", bytesutil.Trunc(deposit.Data.PublicKey))
	}
	if err := beaconState.SetEth1DepositIndex(beaconState.Eth1DepositIndex() + 1); err != nil {
		return nil, err
	}

	return ApplyDeposit(beaconState, deposit.Data, verifySignature)
}

// ApplyDeposit
// Spec pseudocode definition:
// def apply_deposit(state: BeaconState, pubkey: BLSPubkey, withdrawal_credentials: Bytes32, amount: uint64, signature: BLSSignature) -> None:
//
//	validator_pubkeys = [v.pubkey for v in state.validators]
//	if pubkey not in validator_pubkeys:
//	    # Verify the deposit signature (proof of possession) which is not checked by the deposit contract
//	    deposit_message = DepositMessage(
//	        pubkey=pubkey,
//	        withdrawal_credentials=withdrawal_credentials,
//	        amount=amount,
//	    )
//	    domain = compute_domain(DOMAIN_DEPOSIT)  # Fork-agnostic domain since deposits are valid across forks
//	    signing_root = compute_signing_root(deposit_message, domain)
//	    if bls.Verify(pubkey, signing_root, signature):
//	        add_validator_to_registry(state, pubkey, withdrawal_credentials, amount)
//	else:
//	    # Increase balance by deposit amount
//	    index = ValidatorIndex(validator_pubkeys.index(pubkey))
//	    increase_balance(state, index, amount)
func ApplyDeposit(beaconState state.BeaconState, data *ethpb.Deposit_Data, verifySignature bool) (state.BeaconState, error) {
	pubKey := data.PublicKey
	amount := data.Amount
	withdrawalCredentials := data.WithdrawalCredentials
	index, ok := beaconState.ValidatorIndexByPubkey(bytesutil.ToBytes48(pubKey))
	if !ok {
		if verifySignature {
			valid, err := blocks.IsValidDepositSignature(data)
			if err != nil {
				return nil, err
			}
			if !valid {
				return beaconState, nil
			}
		}
		if err := blocks.AddValidatorToRegistry(beaconState, pubKey, withdrawalCredentials, amount); err != nil {
			return nil, err
		}
	} else {
		if err := helpers.IncreaseBalance(beaconState, index, amount); err != nil {
			return nil, err
		}
	}
	return beaconState, nil
}
