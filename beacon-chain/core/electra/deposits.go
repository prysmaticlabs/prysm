package electra

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/contracts/deposit"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// ProcessDeposits is one of the operations performed on each processed
// beacon block to verify queued validators from the Ethereum 1.0 Deposit Contract
// into the beacon chain.
//
// Spec pseudocode definition:
//
//	For each deposit in block.body.deposits:
//	  process_deposit(state, deposit)
func ProcessDeposits(
	ctx context.Context,
	beaconState state.BeaconState,
	deposits []*ethpb.Deposit,
) (state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "electra.ProcessDeposits")
	defer span.End()
	// Attempt to verify all deposit signatures at once, if this fails then fall back to processing
	// individual deposits with signature verification enabled.
	batchVerified, err := blocks.BatchVerifyDepositsSignatures(ctx, deposits)
	if err != nil {
		return nil, errors.Wrap(err, "could not verify deposit signatures in batch")
	}

	for _, d := range deposits {
		if d == nil || d.Data == nil {
			return nil, errors.New("got a nil deposit in block")
		}
		beaconState, err = ProcessDeposit(beaconState, d, batchVerified)
		if err != nil {
			return nil, errors.Wrapf(err, "could not process deposit from %#x", bytesutil.Trunc(d.Data.PublicKey))
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
// def apply_deposit(state: BeaconState, pubkey: BLSPubkey, withdrawal_credentials: Bytes32, amount: uint64, signature: BLSSignature) -> None:
// validator_pubkeys = [v.pubkey for v in state.validators]
// if pubkey not in validator_pubkeys:
//
//	# Verify the deposit signature (proof of possession) which is not checked by the deposit contract
//	if is_valid_deposit_signature(pubkey, withdrawal_credentials, amount, signature):
//	  add_validator_to_registry(state, pubkey, withdrawal_credentials, amount)
//
// else:
//
//	# Increase balance by deposit amount
//	index = ValidatorIndex(validator_pubkeys.index(pubkey))
//	state.pending_balance_deposits.append(PendingBalanceDeposit(index=index, amount=amount))  # [Modified in Electra:EIP-7251]
//	# Check if valid deposit switch to compounding credentials
//
// if ( is_compounding_withdrawal_credential(withdrawal_credentials) and has_eth1_withdrawal_credential(state.validators[index])
//
//	 and is_valid_deposit_signature(pubkey, withdrawal_credentials, amount, signature)
//	):
//	 switch_to_compounding_validator(state, index)
func ApplyDeposit(beaconState state.BeaconState, data *ethpb.Deposit_Data, verifySignature bool) (state.BeaconState, error) {
	pubKey := data.PublicKey
	amount := data.Amount
	withdrawalCredentials := data.WithdrawalCredentials
	index, ok := beaconState.ValidatorIndexByPubkey(bytesutil.ToBytes48(pubKey))
	if !ok {
		if verifySignature {
			valid, err := IsValidDepositSignature(data)
			if err != nil {
				return nil, errors.Wrap(err, "could not verify deposit signature")
			}
			if !valid {
				return beaconState, nil
			}
		}
		if err := AddValidatorToRegistry(beaconState, pubKey, withdrawalCredentials, amount); err != nil {
			return nil, errors.Wrap(err, "could not add validator to registry")
		}
	} else {
		// no validation on top-ups (phase0 feature). no validation before state change
		if err := beaconState.AppendPendingBalanceDeposit(index, amount); err != nil {
			return nil, err
		}
		val, err := beaconState.ValidatorAtIndex(index)
		if err != nil {
			return nil, err
		}
		if helpers.IsCompoundingWithdrawalCredential(withdrawalCredentials) && helpers.HasETH1WithdrawalCredential(val) {
			if verifySignature {
				valid, err := IsValidDepositSignature(data)
				if err != nil {
					return nil, errors.Wrap(err, "could not verify deposit signature")
				}
				if !valid {
					return beaconState, nil
				}
			}
			if err := SwitchToCompoundingValidator(beaconState, index); err != nil {
				return nil, errors.Wrap(err, "could not switch to compound validator")
			}
		}
	}
	return beaconState, nil
}

// IsValidDepositSignature returns whether deposit_data is valid
// def is_valid_deposit_signature(pubkey: BLSPubkey, withdrawal_credentials: Bytes32, amount: uint64, signature: BLSSignature) -> bool:
//
//	deposit_message = DepositMessage( pubkey=pubkey, withdrawal_credentials=withdrawal_credentials, amount=amount, )
//	domain = compute_domain(DOMAIN_DEPOSIT)  # Fork-agnostic domain since deposits are valid across forks
//	signing_root = compute_signing_root(deposit_message, domain)
//	return bls.Verify(pubkey, signing_root, signature)
func IsValidDepositSignature(data *ethpb.Deposit_Data) (bool, error) {
	domain, err := signing.ComputeDomain(params.BeaconConfig().DomainDeposit, nil, nil)
	if err != nil {
		return false, err
	}
	if err := verifyDepositDataSigningRoot(data, domain); err != nil {
		// Ignore this error as in the spec pseudo code.
		log.WithError(err).Debug("Skipping deposit: could not verify deposit data signature")
		return false, nil
	}
	return true, nil
}

func verifyDepositDataSigningRoot(obj *ethpb.Deposit_Data, domain []byte) error {
	return deposit.VerifyDepositSignature(obj, domain)
}

// ProcessPendingBalanceDeposits implements the spec definition below. This method mutates the state.
//
// Spec definition:
//
//	def process_pending_balance_deposits(state: BeaconState) -> None:
//	    available_for_processing = state.deposit_balance_to_consume + get_activation_exit_churn_limit(state)
//	    processed_amount = 0
//	    next_deposit_index = 0
//
//	    for deposit in state.pending_balance_deposits:
//	        if processed_amount + deposit.amount > available_for_processing:
//	            break
//	        increase_balance(state, deposit.index, deposit.amount)
//	        processed_amount += deposit.amount
//	        next_deposit_index += 1
//
//	    state.pending_balance_deposits = state.pending_balance_deposits[next_deposit_index:]
//
//	    if len(state.pending_balance_deposits) == 0:
//	        state.deposit_balance_to_consume = Gwei(0)
//	    else:
//	        state.deposit_balance_to_consume = available_for_processing - processed_amount
func ProcessPendingBalanceDeposits(ctx context.Context, st state.BeaconState, activeBalance primitives.Gwei) error {
	_, span := trace.StartSpan(ctx, "electra.ProcessPendingBalanceDeposits")
	defer span.End()

	if st == nil || st.IsNil() {
		return errors.New("nil state")
	}

	depBalToConsume, err := st.DepositBalanceToConsume()
	if err != nil {
		return err
	}

	availableForProcessing := depBalToConsume + helpers.ActivationExitChurnLimit(activeBalance)
	nextDepositIndex := 0

	deposits, err := st.PendingBalanceDeposits()
	if err != nil {
		return err
	}

	for _, balanceDeposit := range deposits {
		if primitives.Gwei(balanceDeposit.Amount) > availableForProcessing {
			break
		}
		if err := helpers.IncreaseBalance(st, balanceDeposit.Index, balanceDeposit.Amount); err != nil {
			return err
		}
		availableForProcessing -= primitives.Gwei(balanceDeposit.Amount)
		nextDepositIndex++
	}

	deposits = deposits[nextDepositIndex:]
	if err := st.SetPendingBalanceDeposits(deposits); err != nil {
		return err
	}

	if len(deposits) == 0 {
		return st.SetDepositBalanceToConsume(0)
	} else {
		return st.SetDepositBalanceToConsume(availableForProcessing)
	}
}

// ProcessDepositRequests is a function as part of electra to process execution layer deposits
func ProcessDepositRequests(ctx context.Context, beaconState state.BeaconState, requests []*enginev1.DepositRequest) (state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "electra.ProcessDepositRequests")
	defer span.End()

	if len(requests) == 0 {
		log.Debug("ProcessDepositRequests: no deposit requests found")
		return beaconState, nil
	}

	deposits := make([]*ethpb.Deposit, 0)
	for _, req := range requests {
		if req == nil {
			return nil, errors.New("got a nil DepositRequest")
		}
		deposits = append(deposits, &ethpb.Deposit{
			Data: &ethpb.Deposit_Data{
				PublicKey:             req.Pubkey,
				WithdrawalCredentials: req.WithdrawalCredentials,
				Amount:                req.Amount,
				Signature:             req.Signature,
			},
		})
	}
	batchVerified, err := blocks.BatchVerifyDepositsSignatures(ctx, deposits)
	if err != nil {
		return nil, errors.Wrap(err, "could not verify deposit signatures in batch")
	}
	for _, receipt := range requests {
		beaconState, err = processDepositRequest(beaconState, receipt, batchVerified)
		if err != nil {
			return nil, errors.Wrap(err, "could not apply deposit request")
		}
	}
	return beaconState, nil
}

// processDepositRequest processes the specific deposit receipt
// def process_deposit_request(state: BeaconState, deposit_request: DepositRequest) -> None:
//
//	# Set deposit request start index
//	if state.deposit_requests_start_index == UNSET_DEPOSIT_REQUEST_START_INDEX:
//	    state.deposit_requests_start_index = deposit_request.index
//
//	apply_deposit(
//	    state=state,
//	    pubkey=deposit_request.pubkey,
//	    withdrawal_credentials=deposit_request.withdrawal_credentials,
//	    amount=deposit_request.amount,
//	    signature=deposit_request.signature,
//	)
func processDepositRequest(beaconState state.BeaconState, request *enginev1.DepositRequest, verifySignature bool) (state.BeaconState, error) {
	requestsStartIndex, err := beaconState.DepositRequestsStartIndex()
	if err != nil {
		return nil, errors.Wrap(err, "could not get deposit requests start index")
	}
	if requestsStartIndex == params.BeaconConfig().UnsetDepositRequestsStartIndex {
		if err := beaconState.SetDepositRequestsStartIndex(request.Index); err != nil {
			return nil, errors.Wrap(err, "could not set deposit requests start index")
		}
	}
	return ApplyDeposit(beaconState, &ethpb.Deposit_Data{
		PublicKey:             bytesutil.SafeCopyBytes(request.Pubkey),
		Amount:                request.Amount,
		WithdrawalCredentials: bytesutil.SafeCopyBytes(request.WithdrawalCredentials),
		Signature:             bytesutil.SafeCopyBytes(request.Signature),
	}, verifySignature)
}
