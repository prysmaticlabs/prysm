// Package client represents a gRPC polling-based implementation
// of an Ethereum validator client.
package client

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	lru "github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/async/event"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/v3/config/features"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	validatorserviceconfig "github.com/prysmaticlabs/prysm/v3/config/validator/service"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/crypto/hash"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	accountsiface "github.com/prysmaticlabs/prysm/v3/validator/accounts/iface"
	"github.com/prysmaticlabs/prysm/v3/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/v3/validator/client/iface"
	vdb "github.com/prysmaticlabs/prysm/v3/validator/db"
	"github.com/prysmaticlabs/prysm/v3/validator/db/kv"
	"github.com/prysmaticlabs/prysm/v3/validator/graffiti"
	"github.com/prysmaticlabs/prysm/v3/validator/keymanager"
	"github.com/prysmaticlabs/prysm/v3/validator/keymanager/local"
	remoteweb3signer "github.com/prysmaticlabs/prysm/v3/validator/keymanager/remote-web3signer"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

// keyFetchPeriod is the frequency that we try to refetch validating keys
// in case no keys were fetched previously.
var (
	keyRefetchPeriod                = 30 * time.Second
	ErrBuilderValidatorRegistration = errors.New("Builder API validator registration unsuccessful")
)

var (
	msgCouldNotFetchKeys = "could not fetch validating keys"
	msgNoKeysFetched     = "No validating keys fetched. Trying again"
)

type validator struct {
	logValidatorBalances               bool
	useWeb                             bool
	emitAccountMetrics                 bool
	domainDataLock                     sync.Mutex
	attLogsLock                        sync.Mutex
	aggregatedSlotCommitteeIDCacheLock sync.Mutex
	highestValidSlotLock               sync.Mutex
	prevBalanceLock                    sync.RWMutex
	slashableKeysLock                  sync.RWMutex
	eipImportBlacklistedPublicKeys     map[[fieldparams.BLSPubkeyLength]byte]bool
	walletInitializedFeed              *event.Feed
	attLogs                            map[[32]byte]*attSubmitted
	startBalances                      map[[fieldparams.BLSPubkeyLength]byte]uint64
	duties                             *ethpb.DutiesResponse
	prevBalance                        map[[fieldparams.BLSPubkeyLength]byte]uint64
	pubkeyToValidatorIndex             map[[fieldparams.BLSPubkeyLength]byte]types.ValidatorIndex
	signedValidatorRegistrations       map[[fieldparams.BLSPubkeyLength]byte]*ethpb.SignedValidatorRegistrationV1
	graffitiOrderedIndex               uint64
	aggregatedSlotCommitteeIDCache     *lru.Cache
	domainDataCache                    *ristretto.Cache
	highestValidSlot                   types.Slot
	genesisTime                        uint64
	blockFeed                          *event.Feed
	interopKeysConfig                  *local.InteropKeymanagerConfig
	wallet                             *wallet.Wallet
	graffitiStruct                     *graffiti.Graffiti
	node                               ethpb.NodeClient
	slashingProtectionClient           ethpb.SlasherClient
	db                                 vdb.Database
	beaconClient                       ethpb.BeaconChainClient
	keyManager                         keymanager.IKeymanager
	ticker                             slots.Ticker
	validatorClient                    ethpb.BeaconNodeValidatorClient
	graffiti                           []byte
	voteStats                          voteStats
	syncCommitteeStats                 syncCommitteeStats
	Web3SignerConfig                   *remoteweb3signer.SetupConfig
	ProposerSettings                   *validatorserviceconfig.ProposerSettings
	walletInitializedChannel           chan *wallet.Wallet
}

type validatorStatus struct {
	publicKey []byte
	status    *ethpb.ValidatorStatusResponse
	index     types.ValidatorIndex
}

// Done cleans up the validator.
func (v *validator) Done() {
	v.ticker.Done()
}

// WaitForKeymanagerInitialization checks if the validator needs to wait for
func (v *validator) WaitForKeymanagerInitialization(ctx context.Context) error {
	genesisRoot, err := v.db.GenesisValidatorsRoot(ctx)
	if err != nil {
		return errors.Wrap(err, "unable to retrieve valid genesis validators root while initializing key manager")
	}

	if v.useWeb && v.wallet == nil {
		log.Info("Waiting for keymanager to initialize validator client with web UI")
		// if wallet is not set, wait for it to be set through the UI
		km, err := waitForWebWalletInitialization(ctx, v.walletInitializedFeed, v.walletInitializedChannel)
		if err != nil {
			return err
		}
		v.keyManager = km
	} else {
		if v.interopKeysConfig != nil {
			keyManager, err := local.NewInteropKeymanager(ctx, v.interopKeysConfig.Offset, v.interopKeysConfig.NumValidatorKeys)
			if err != nil {
				return errors.Wrap(err, "could not generate interop keys for key manager")
			}
			v.keyManager = keyManager
		} else if v.wallet == nil {
			return errors.New("wallet not set")
		} else {
			if v.Web3SignerConfig != nil {
				v.Web3SignerConfig.GenesisValidatorsRoot = genesisRoot
			}
			keyManager, err := v.wallet.InitializeKeymanager(ctx, accountsiface.InitKeymanagerConfig{ListenForChanges: true, Web3SignerConfig: v.Web3SignerConfig})
			if err != nil {
				return errors.Wrap(err, "could not initialize key manager")
			}
			v.keyManager = keyManager
		}
	}
	recheckKeys(ctx, v.db, v.keyManager)
	return nil
}

// subscribe to channel for when the wallet is initialized
func waitForWebWalletInitialization(
	ctx context.Context,
	walletInitializedEvent *event.Feed,
	walletChan chan *wallet.Wallet,
) (keymanager.IKeymanager, error) {
	sub := walletInitializedEvent.Subscribe(walletChan)
	defer sub.Unsubscribe()
	for {
		select {
		case w := <-walletChan:
			keyManager, err := w.InitializeKeymanager(ctx, accountsiface.InitKeymanagerConfig{ListenForChanges: true})
			if err != nil {
				return nil, errors.Wrap(err, "could not read keymanager")
			}
			return keyManager, nil
		case <-ctx.Done():
			return nil, errors.New("context canceled")
		case <-sub.Err():
			log.Error("Subscriber closed, exiting goroutine")
			return nil, nil
		}
	}
}

// recheckKeys checks if the validator has any keys that need to be rechecked.
// the keymanager implements a subscription to push these updates to the validator.
func recheckKeys(ctx context.Context, valDB vdb.Database, keyManager keymanager.IKeymanager) {
	var validatingKeys [][fieldparams.BLSPubkeyLength]byte
	var err error
	validatingKeys, err = keyManager.FetchValidatingPublicKeys(ctx)
	if err != nil {
		log.WithError(err).Debug("Could not fetch validating keys")
	}
	if err := valDB.UpdatePublicKeysBuckets(validatingKeys); err != nil {
		log.WithError(err).Debug("Could not update public keys buckets")
	}
	go recheckValidatingKeysBucket(ctx, valDB, keyManager)
	for _, key := range validatingKeys {
		log.WithField(
			"publicKey", fmt.Sprintf("%#x", bytesutil.Trunc(key[:])),
		).Info("Validating for public key")
	}
}

// to accounts changes in the keymanager, then updates those keys'
// buckets in bolt DB if a bucket for a key does not exist.
func recheckValidatingKeysBucket(ctx context.Context, valDB vdb.Database, km keymanager.IKeymanager) {
	importedKeymanager, ok := km.(*local.Keymanager)
	if !ok {
		return
	}
	validatingPubKeysChan := make(chan [][fieldparams.BLSPubkeyLength]byte, 1)
	sub := importedKeymanager.SubscribeAccountChanges(validatingPubKeysChan)
	defer func() {
		sub.Unsubscribe()
		close(validatingPubKeysChan)
	}()
	for {
		select {
		case keys := <-validatingPubKeysChan:
			if err := valDB.UpdatePublicKeysBuckets(keys); err != nil {
				log.WithError(err).Debug("Could not update public keys buckets")
				continue
			}
		case <-ctx.Done():
			return
		case <-sub.Err():
			log.Error("Subscriber closed, exiting goroutine")
			return
		}
	}
}

// WaitForChainStart checks whether the beacon node has started its runtime. That is,
// it calls to the beacon node which then verifies the ETH1.0 deposit contract logs to check
// for the ChainStart log to have been emitted. If so, it starts a ticker based on the ChainStart
// unix timestamp which will be used to keep track of time within the validator client.
func (v *validator) WaitForChainStart(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "validator.WaitForChainStart")
	defer span.End()
	// First, check if the beacon chain has started.
	stream, err := v.validatorClient.WaitForChainStart(ctx, &emptypb.Empty{})
	if err != nil {
		return errors.Wrap(
			iface.ErrConnectionIssue,
			errors.Wrap(err, "could not setup beacon chain ChainStart streaming client").Error(),
		)
	}

	log.Info("Syncing with beacon node to align on chain genesis info")
	chainStartRes, err := stream.Recv()
	if err != io.EOF {
		if ctx.Err() == context.Canceled {
			return errors.Wrap(ctx.Err(), "context has been canceled so shutting down the loop")
		}
		if err != nil {
			return errors.Wrap(
				iface.ErrConnectionIssue,
				errors.Wrap(err, "could not receive ChainStart from stream").Error(),
			)
		}
		v.genesisTime = chainStartRes.GenesisTime
		curGenValRoot, err := v.db.GenesisValidatorsRoot(ctx)
		if err != nil {
			return errors.Wrap(err, "could not get current genesis validators root")
		}
		if len(curGenValRoot) == 0 {
			if err := v.db.SaveGenesisValidatorsRoot(ctx, chainStartRes.GenesisValidatorsRoot); err != nil {
				return errors.Wrap(err, "could not save genesis validators root")
			}
		} else {
			if !bytes.Equal(curGenValRoot, chainStartRes.GenesisValidatorsRoot) {
				log.Errorf("The genesis validators root received from the beacon node does not match what is in " +
					"your validator database. This could indicate that this is a database meant for another network. If " +
					"you were previously running this validator database on another network, please run --clear-db to " +
					"clear the database. If not, please file an issue at https://github.com/prysmaticlabs/prysm/issues")
				return fmt.Errorf(
					"genesis validators root from beacon node (%#x) does not match root saved in validator db (%#x)",
					chainStartRes.GenesisValidatorsRoot,
					curGenValRoot,
				)
			}
		}
	} else {
		return iface.ErrConnectionIssue
	}

	// Once the ChainStart log is received, we update the genesis time of the validator client
	// and begin a slot ticker used to track the current slot the beacon node is in.
	v.ticker = slots.NewSlotTicker(time.Unix(int64(v.genesisTime), 0), params.BeaconConfig().SecondsPerSlot)
	log.WithField("genesisTime", time.Unix(int64(v.genesisTime), 0)).Info("Beacon chain started")
	return nil
}

// WaitForSync checks whether the beacon node has sync to the latest head.
func (v *validator) WaitForSync(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "validator.WaitForSync")
	defer span.End()

	s, err := v.node.GetSyncStatus(ctx, &emptypb.Empty{})
	if err != nil {
		return errors.Wrap(iface.ErrConnectionIssue, errors.Wrap(err, "could not get sync status").Error())
	}
	if !s.Syncing {
		return nil
	}

	for {
		select {
		// Poll every half slot.
		case <-time.After(slots.DivideSlotBy(2 /* twice per slot */)):
			s, err := v.node.GetSyncStatus(ctx, &emptypb.Empty{})
			if err != nil {
				return errors.Wrap(iface.ErrConnectionIssue, errors.Wrap(err, "could not get sync status").Error())
			}
			if !s.Syncing {
				return nil
			}
			log.Info("Waiting for beacon node to sync to latest chain head")
		case <-ctx.Done():
			return errors.New("context has been canceled, exiting goroutine")
		}
	}
}

// ReceiveBlocks starts a gRPC client stream listener to obtain
// blocks from the beacon node. Upon receiving a block, the service
// broadcasts it to a feed for other usages to subscribe to.
func (v *validator) ReceiveBlocks(ctx context.Context, connectionErrorChannel chan<- error) {
	stream, err := v.validatorClient.StreamBlocksAltair(ctx, &ethpb.StreamBlocksRequest{VerifiedOnly: true})
	if err != nil {
		log.WithError(err).Error("Failed to retrieve blocks stream, " + iface.ErrConnectionIssue.Error())
		connectionErrorChannel <- errors.Wrap(iface.ErrConnectionIssue, err.Error())
		return
	}

	for {
		if ctx.Err() == context.Canceled {
			log.WithError(ctx.Err()).Error("Context canceled - shutting down blocks receiver")
			return
		}
		res, err := stream.Recv()
		if err != nil {
			log.WithError(err).Error("Could not receive blocks from beacon node, " + iface.ErrConnectionIssue.Error())
			connectionErrorChannel <- errors.Wrap(iface.ErrConnectionIssue, err.Error())
			return
		}
		if res == nil || res.Block == nil {
			continue
		}
		var blk interfaces.SignedBeaconBlock
		switch b := res.Block.(type) {
		case *ethpb.StreamBlocksResponse_Phase0Block:
			blk, err = blocks.NewSignedBeaconBlock(b.Phase0Block)
		case *ethpb.StreamBlocksResponse_AltairBlock:
			blk, err = blocks.NewSignedBeaconBlock(b.AltairBlock)
		case *ethpb.StreamBlocksResponse_BellatrixBlock:
			blk, err = blocks.NewSignedBeaconBlock(b.BellatrixBlock)
		}
		if err != nil {
			log.WithError(err).Error("Failed to wrap signed block")
			continue
		}
		if blk == nil || blk.IsNil() {
			log.Error("Received nil block")
			continue
		}
		v.highestValidSlotLock.Lock()
		if blk.Block().Slot() > v.highestValidSlot {
			v.highestValidSlot = blk.Block().Slot()
		}
		v.highestValidSlotLock.Unlock()
		v.blockFeed.Send(blk)
	}
}

func (v *validator) checkAndLogValidatorStatus(statuses []*validatorStatus, activeValCount uint64) bool {
	activationsPerEpoch :=
		uint64(math.Max(float64(params.BeaconConfig().MinPerEpochChurnLimit), float64(activeValCount/params.BeaconConfig().ChurnLimitQuotient)))

	nonexistentIndex := types.ValidatorIndex(^uint64(0))
	var validatorActivated bool
	for _, status := range statuses {
		fields := logrus.Fields{
			"pubKey": fmt.Sprintf("%#x", bytesutil.Trunc(status.publicKey)),
			"status": status.status.Status.String(),
		}
		if status.index != nonexistentIndex {
			fields["index"] = status.index
		}
		log := log.WithFields(fields)
		if v.emitAccountMetrics {
			fmtKey := fmt.Sprintf("%#x", status.publicKey)
			ValidatorStatusesGaugeVec.WithLabelValues(fmtKey).Set(float64(status.status.Status))
		}
		switch status.status.Status {
		case ethpb.ValidatorStatus_UNKNOWN_STATUS:
			log.Info("Waiting for deposit to be observed by beacon node")
		case ethpb.ValidatorStatus_DEPOSITED:
			if status.status.PositionInActivationQueue != 0 {
				log.WithField(
					"positionInActivationQueue", status.status.PositionInActivationQueue,
				).Info("Deposit processed, entering activation queue after finalization")
			}
		case ethpb.ValidatorStatus_PENDING:
			secondsPerEpoch := uint64(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().SecondsPerSlot))
			expectedWaitingTime :=
				time.Duration((status.status.PositionInActivationQueue+activationsPerEpoch)/activationsPerEpoch*secondsPerEpoch) * time.Second
			if status.status.ActivationEpoch == params.BeaconConfig().FarFutureEpoch {
				log.WithFields(logrus.Fields{
					"positionInActivationQueue": status.status.PositionInActivationQueue,
					"expectedWaitingTime":       expectedWaitingTime.String(),
				}).Info("Waiting to be assigned activation epoch")
			} else {
				log.WithFields(logrus.Fields{
					"activationEpoch": status.status.ActivationEpoch,
				}).Info("Waiting for activation")
			}
		case ethpb.ValidatorStatus_ACTIVE, ethpb.ValidatorStatus_EXITING:
			validatorActivated = true
		case ethpb.ValidatorStatus_EXITED:
			log.Info("Validator exited")
		case ethpb.ValidatorStatus_INVALID:
			log.Warn("Invalid Eth1 deposit")
		default:
			log.WithFields(logrus.Fields{
				"activationEpoch": status.status.ActivationEpoch,
			}).Info("Validator status")
		}
	}
	return validatorActivated
}

func logActiveValidatorStatus(statuses []*validatorStatus) {
	for _, s := range statuses {
		if s.status.Status != ethpb.ValidatorStatus_ACTIVE {
			continue
		}
		log.WithFields(logrus.Fields{
			"publicKey": fmt.Sprintf("%#x", bytesutil.Trunc(s.publicKey)),
			"index":     s.index,
		}).Info("Validator activated")
	}
}

// CanonicalHeadSlot returns the slot of canonical block currently found in the
// beacon chain via RPC.
func (v *validator) CanonicalHeadSlot(ctx context.Context) (types.Slot, error) {
	ctx, span := trace.StartSpan(ctx, "validator.CanonicalHeadSlot")
	defer span.End()
	head, err := v.beaconClient.GetChainHead(ctx, &emptypb.Empty{})
	if err != nil {
		return 0, errors.Wrap(iface.ErrConnectionIssue, err.Error())
	}
	return head.HeadSlot, nil
}

// NextSlot emits the next slot number at the start time of that slot.
func (v *validator) NextSlot() <-chan types.Slot {
	return v.ticker.C()
}

// SlotDeadline is the start time of the next slot.
func (v *validator) SlotDeadline(slot types.Slot) time.Time {
	secs := time.Duration((slot + 1).Mul(params.BeaconConfig().SecondsPerSlot))
	return time.Unix(int64(v.genesisTime), 0 /*ns*/).Add(secs * time.Second)
}

// CheckDoppelGanger checks if the current actively provided keys have
// any duplicates active in the network.
func (v *validator) CheckDoppelGanger(ctx context.Context) error {
	if !features.Get().EnableDoppelGanger {
		return nil
	}
	pubkeys, err := v.keyManager.FetchValidatingPublicKeys(ctx)
	if err != nil {
		return err
	}
	log.WithField("keys", len(pubkeys)).Info("Running doppelganger check")
	// Exit early if no validating pub keys are found.
	if len(pubkeys) == 0 {
		return nil
	}
	req := &ethpb.DoppelGangerRequest{ValidatorRequests: []*ethpb.DoppelGangerRequest_ValidatorRequest{}}
	for _, pkey := range pubkeys {
		copiedKey := pkey
		attRec, err := v.db.AttestationHistoryForPubKey(ctx, copiedKey)
		if err != nil {
			return err
		}
		if len(attRec) == 0 {
			// If no history exists we simply send in a zero
			// value for the request epoch and root.
			req.ValidatorRequests = append(req.ValidatorRequests,
				&ethpb.DoppelGangerRequest_ValidatorRequest{
					PublicKey:  copiedKey[:],
					Epoch:      0,
					SignedRoot: make([]byte, fieldparams.RootLength),
				})
			continue
		}
		r := retrieveLatestRecord(attRec)
		if copiedKey != r.PubKey {
			return errors.New("attestation record mismatched public key")
		}
		req.ValidatorRequests = append(req.ValidatorRequests,
			&ethpb.DoppelGangerRequest_ValidatorRequest{
				PublicKey:  r.PubKey[:],
				Epoch:      r.Target,
				SignedRoot: r.SigningRoot[:],
			})
	}
	resp, err := v.validatorClient.CheckDoppelGanger(ctx, req)
	if err != nil {
		return err
	}
	// If nothing is returned by the beacon node, we return an
	// error as it is unsafe for us to proceed.
	if resp == nil || resp.Responses == nil || len(resp.Responses) == 0 {
		return errors.New("beacon node returned 0 responses for doppelganger check")
	}
	return buildDuplicateError(resp.Responses)
}

func buildDuplicateError(response []*ethpb.DoppelGangerResponse_ValidatorResponse) error {
	duplicates := make([][]byte, 0)
	for _, valRes := range response {
		if valRes.DuplicateExists {
			copiedKey := [fieldparams.BLSPubkeyLength]byte{}
			copy(copiedKey[:], valRes.PublicKey)
			duplicates = append(duplicates, copiedKey[:])
		}
	}
	if len(duplicates) == 0 {
		return nil
	}
	return errors.Errorf("Duplicate instances exists in the network for validator keys: %#x", duplicates)
}

// Ensures that the latest attestation history is retrieved.
func retrieveLatestRecord(recs []*kv.AttestationRecord) *kv.AttestationRecord {
	if len(recs) == 0 {
		return nil
	}
	lastSource := recs[len(recs)-1].Source
	chosenRec := recs[len(recs)-1]
	for i := len(recs) - 1; i >= 0; i-- {
		// Exit if we are now on a different source
		// as it is assumed that all source records are
		// byte sorted.
		if recs[i].Source != lastSource {
			break
		}
		// If we have a smaller target, we do
		// change our chosen record.
		if chosenRec.Target < recs[i].Target {
			chosenRec = recs[i]
		}
	}
	return chosenRec
}

// UpdateDuties checks the slot number to determine if the validator's
// list of upcoming assignments needs to be updated. For example, at the
// beginning of a new epoch.
func (v *validator) UpdateDuties(ctx context.Context, slot types.Slot) error {
	if slot%params.BeaconConfig().SlotsPerEpoch != 0 && v.duties != nil {
		// Do nothing if not epoch start AND assignments already exist.
		return nil
	}
	// Set deadline to end of epoch.
	ss, err := slots.EpochStart(slots.ToEpoch(slot) + 1)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithDeadline(ctx, v.SlotDeadline(ss))
	defer cancel()
	ctx, span := trace.StartSpan(ctx, "validator.UpdateAssignments")
	defer span.End()

	validatingKeys, err := v.keyManager.FetchValidatingPublicKeys(ctx)
	if err != nil {
		return err
	}

	// Filter out the slashable public keys from the duties request.
	filteredKeys := make([][fieldparams.BLSPubkeyLength]byte, 0, len(validatingKeys))
	v.slashableKeysLock.RLock()
	for _, pubKey := range validatingKeys {
		if ok := v.eipImportBlacklistedPublicKeys[pubKey]; !ok {
			filteredKeys = append(filteredKeys, pubKey)
		} else {
			log.WithField(
				"publicKey", fmt.Sprintf("%#x", bytesutil.Trunc(pubKey[:])),
			).Warn("Not including slashable public key from slashing protection import " +
				"in request to update validator duties")
		}
	}
	v.slashableKeysLock.RUnlock()

	req := &ethpb.DutiesRequest{
		Epoch:      types.Epoch(slot / params.BeaconConfig().SlotsPerEpoch),
		PublicKeys: bytesutil.FromBytes48Array(filteredKeys),
	}

	// If duties is nil it means we have had no prior duties and just started up.
	resp, err := v.validatorClient.GetDuties(ctx, req)
	if err != nil {
		v.duties = nil // Clear assignments so we know to retry the request.
		log.Error(err)
		return err
	}

	v.duties = resp
	v.logDuties(slot, v.duties.CurrentEpochDuties)

	// Non-blocking call for beacon node to start subscriptions for aggregators.
	go func() {
		if err := v.subscribeToSubnets(context.Background(), resp); err != nil {
			log.WithError(err).Error("Failed to subscribe to subnets")
		}
	}()

	return nil
}

// subscribeToSubnets iterates through each validator duty, signs each slot, and asks beacon node
// to eagerly subscribe to subnets so that the aggregator has attestations to aggregate.
func (v *validator) subscribeToSubnets(ctx context.Context, res *ethpb.DutiesResponse) error {
	subscribeSlots := make([]types.Slot, 0, len(res.CurrentEpochDuties)+len(res.NextEpochDuties))
	subscribeCommitteeIndices := make([]types.CommitteeIndex, 0, len(res.CurrentEpochDuties)+len(res.NextEpochDuties))
	subscribeIsAggregator := make([]bool, 0, len(res.CurrentEpochDuties)+len(res.NextEpochDuties))
	alreadySubscribed := make(map[[64]byte]bool)

	for _, duty := range res.CurrentEpochDuties {
		pk := bytesutil.ToBytes48(duty.PublicKey)
		if duty.Status == ethpb.ValidatorStatus_ACTIVE || duty.Status == ethpb.ValidatorStatus_EXITING {
			attesterSlot := duty.AttesterSlot
			committeeIndex := duty.CommitteeIndex

			alreadySubscribedKey := validatorSubscribeKey(attesterSlot, committeeIndex)
			if _, ok := alreadySubscribed[alreadySubscribedKey]; ok {
				continue
			}

			aggregator, err := v.isAggregator(ctx, duty.Committee, attesterSlot, pk)
			if err != nil {
				return errors.Wrap(err, "could not check if a validator is an aggregator")
			}
			if aggregator {
				alreadySubscribed[alreadySubscribedKey] = true
			}

			subscribeSlots = append(subscribeSlots, attesterSlot)
			subscribeCommitteeIndices = append(subscribeCommitteeIndices, committeeIndex)
			subscribeIsAggregator = append(subscribeIsAggregator, aggregator)
		}
	}

	for _, duty := range res.NextEpochDuties {
		if duty.Status == ethpb.ValidatorStatus_ACTIVE || duty.Status == ethpb.ValidatorStatus_EXITING {
			attesterSlot := duty.AttesterSlot
			committeeIndex := duty.CommitteeIndex

			alreadySubscribedKey := validatorSubscribeKey(attesterSlot, committeeIndex)
			if _, ok := alreadySubscribed[alreadySubscribedKey]; ok {
				continue
			}

			aggregator, err := v.isAggregator(ctx, duty.Committee, attesterSlot, bytesutil.ToBytes48(duty.PublicKey))
			if err != nil {
				return errors.Wrap(err, "could not check if a validator is an aggregator")
			}
			if aggregator {
				alreadySubscribed[alreadySubscribedKey] = true
			}

			subscribeSlots = append(subscribeSlots, attesterSlot)
			subscribeCommitteeIndices = append(subscribeCommitteeIndices, committeeIndex)
			subscribeIsAggregator = append(subscribeIsAggregator, aggregator)
		}
	}

	_, err := v.validatorClient.SubscribeCommitteeSubnets(ctx, &ethpb.CommitteeSubnetsSubscribeRequest{
		Slots:        subscribeSlots,
		CommitteeIds: subscribeCommitteeIndices,
		IsAggregator: subscribeIsAggregator,
	})

	return err
}

// RolesAt slot returns the validator roles at the given slot. Returns nil if the
// validator is known to not have a roles at the slot. Returns UNKNOWN if the
// validator assignments are unknown. Otherwise returns a valid ValidatorRole map.
func (v *validator) RolesAt(ctx context.Context, slot types.Slot) (map[[fieldparams.BLSPubkeyLength]byte][]iface.ValidatorRole, error) {
	rolesAt := make(map[[fieldparams.BLSPubkeyLength]byte][]iface.ValidatorRole)
	for validator, duty := range v.duties.Duties {
		var roles []iface.ValidatorRole

		if duty == nil {
			continue
		}
		if len(duty.ProposerSlots) > 0 {
			for _, proposerSlot := range duty.ProposerSlots {
				if proposerSlot != 0 && proposerSlot == slot {
					roles = append(roles, iface.RoleProposer)
					break
				}
			}
		}
		if duty.AttesterSlot == slot {
			roles = append(roles, iface.RoleAttester)

			aggregator, err := v.isAggregator(ctx, duty.Committee, slot, bytesutil.ToBytes48(duty.PublicKey))
			if err != nil {
				return nil, errors.Wrap(err, "could not check if a validator is an aggregator")
			}
			if aggregator {
				roles = append(roles, iface.RoleAggregator)
			}

		}

		// Being assigned to a sync committee for a given slot means that the validator produces and
		// broadcasts signatures for `slot - 1` for inclusion in `slot`. At the last slot of the epoch,
		// the validator checks whether it's in the sync committee of following epoch.
		inSyncCommittee := false
		if slots.IsEpochEnd(slot) {
			if v.duties.NextEpochDuties[validator].IsSyncCommittee {
				roles = append(roles, iface.RoleSyncCommittee)
				inSyncCommittee = true
			}
		} else {
			if duty.IsSyncCommittee {
				roles = append(roles, iface.RoleSyncCommittee)
				inSyncCommittee = true
			}
		}
		if inSyncCommittee {
			aggregator, err := v.isSyncCommitteeAggregator(ctx, slot, bytesutil.ToBytes48(duty.PublicKey))
			if err != nil {
				return nil, errors.Wrap(err, "could not check if a validator is a sync committee aggregator")
			}
			if aggregator {
				roles = append(roles, iface.RoleSyncCommitteeAggregator)
			}
		}

		if len(roles) == 0 {
			roles = append(roles, iface.RoleUnknown)
		}

		var pubKey [fieldparams.BLSPubkeyLength]byte
		copy(pubKey[:], duty.PublicKey)
		rolesAt[pubKey] = roles
	}
	return rolesAt, nil
}

// Keymanager returns the underlying validator's keymanager.
func (v *validator) Keymanager() (keymanager.IKeymanager, error) {
	if v.keyManager == nil {
		return nil, errors.New("keymanager is not initialized")
	}
	return v.keyManager, nil
}

// isAggregator checks if a validator is an aggregator of a given slot and committee,
// it uses a modulo calculated by validator count in committee and samples randomness around it.
func (v *validator) isAggregator(ctx context.Context, committee []types.ValidatorIndex, slot types.Slot, pubKey [fieldparams.BLSPubkeyLength]byte) (bool, error) {
	modulo := uint64(1)
	if len(committee)/int(params.BeaconConfig().TargetAggregatorsPerCommittee) > 1 {
		modulo = uint64(len(committee)) / params.BeaconConfig().TargetAggregatorsPerCommittee
	}

	slotSig, err := v.signSlotWithSelectionProof(ctx, pubKey, slot)
	if err != nil {
		return false, err
	}

	b := hash.Hash(slotSig)

	return binary.LittleEndian.Uint64(b[:8])%modulo == 0, nil
}

// isSyncCommitteeAggregator checks if a validator in an aggregator of a subcommittee for sync committee.
// it uses a modulo calculated by validator count in committee and samples randomness around it.
//
// Spec code:
// def is_sync_committee_aggregator(signature: BLSSignature) -> bool:
//    modulo = max(1, SYNC_COMMITTEE_SIZE // SYNC_COMMITTEE_SUBNET_COUNT // TARGET_AGGREGATORS_PER_SYNC_SUBCOMMITTEE)
//    return bytes_to_uint64(hash(signature)[0:8]) % modulo == 0
func (v *validator) isSyncCommitteeAggregator(ctx context.Context, slot types.Slot, pubKey [fieldparams.BLSPubkeyLength]byte) (bool, error) {
	res, err := v.validatorClient.GetSyncSubcommitteeIndex(ctx, &ethpb.SyncSubcommitteeIndexRequest{
		PublicKey: pubKey[:],
		Slot:      slot,
	})
	if err != nil {
		return false, err
	}

	for _, index := range res.Indices {
		subCommitteeSize := params.BeaconConfig().SyncCommitteeSize / params.BeaconConfig().SyncCommitteeSubnetCount
		subnet := uint64(index) / subCommitteeSize
		sig, err := v.signSyncSelectionData(ctx, pubKey, subnet, slot)
		if err != nil {
			return false, err
		}
		isAggregator, err := altair.IsSyncCommitteeAggregator(sig)
		if err != nil {
			return false, err
		}
		if isAggregator {
			return true, nil
		}
	}

	return false, nil
}

// UpdateDomainDataCaches by making calls for all of the possible domain data. These can change when
// the fork version changes which can happen once per epoch. Although changing for the fork version
// is very rare, a validator should check these data every epoch to be sure the validator is
// participating on the correct fork version.
func (v *validator) UpdateDomainDataCaches(ctx context.Context, slot types.Slot) {
	for _, d := range [][]byte{
		params.BeaconConfig().DomainRandao[:],
		params.BeaconConfig().DomainBeaconAttester[:],
		params.BeaconConfig().DomainBeaconProposer[:],
		params.BeaconConfig().DomainSelectionProof[:],
		params.BeaconConfig().DomainAggregateAndProof[:],
	} {
		_, err := v.domainData(ctx, slots.ToEpoch(slot), d)
		if err != nil {
			log.WithError(err).Errorf("Failed to update domain data for domain %v", d)
		}
	}
}

// AllValidatorsAreExited informs whether all validators have already exited.
func (v *validator) AllValidatorsAreExited(ctx context.Context) (bool, error) {
	validatingKeys, err := v.keyManager.FetchValidatingPublicKeys(ctx)
	if err != nil {
		return false, errors.Wrap(err, "could not fetch validating keys")
	}
	if len(validatingKeys) == 0 {
		return false, nil
	}
	var publicKeys [][]byte
	for _, key := range validatingKeys {
		copyKey := key
		publicKeys = append(publicKeys, copyKey[:])
	}
	request := &ethpb.MultipleValidatorStatusRequest{
		PublicKeys: publicKeys,
	}
	response, err := v.validatorClient.MultipleValidatorStatus(ctx, request)
	if err != nil {
		return false, err
	}
	if len(response.Statuses) != len(request.PublicKeys) {
		return false, errors.New("number of status responses did not match number of requested keys")
	}
	for _, status := range response.Statuses {
		if status.Status != ethpb.ValidatorStatus_EXITED {
			return false, nil
		}
	}
	return true, nil
}

func (v *validator) domainData(ctx context.Context, epoch types.Epoch, domain []byte) (*ethpb.DomainResponse, error) {
	v.domainDataLock.Lock()
	defer v.domainDataLock.Unlock()

	req := &ethpb.DomainRequest{
		Epoch:  epoch,
		Domain: domain,
	}

	key := strings.Join([]string{strconv.FormatUint(uint64(req.Epoch), 10), hex.EncodeToString(req.Domain)}, ",")

	if val, ok := v.domainDataCache.Get(key); ok {
		return proto.Clone(val.(proto.Message)).(*ethpb.DomainResponse), nil
	}

	res, err := v.validatorClient.DomainData(ctx, req)
	if err != nil {
		return nil, err
	}

	v.domainDataCache.Set(key, proto.Clone(res), 1)

	return res, nil
}

func (v *validator) logDuties(slot types.Slot, duties []*ethpb.DutiesResponse_Duty) {
	attesterKeys := make([][]string, params.BeaconConfig().SlotsPerEpoch)
	for i := range attesterKeys {
		attesterKeys[i] = make([]string, 0)
	}
	proposerKeys := make([]string, params.BeaconConfig().SlotsPerEpoch)
	slotOffset := slot - (slot % params.BeaconConfig().SlotsPerEpoch)
	var totalAttestingKeys uint64
	for _, duty := range duties {
		validatorNotTruncatedKey := fmt.Sprintf("%#x", duty.PublicKey)
		if v.emitAccountMetrics {
			ValidatorStatusesGaugeVec.WithLabelValues(validatorNotTruncatedKey).Set(float64(duty.Status))
		}

		// Only interested in validators who are attesting/proposing.
		// Note that SLASHING validators will have duties but their results are ignored by the network so we don't bother with them.
		if duty.Status != ethpb.ValidatorStatus_ACTIVE && duty.Status != ethpb.ValidatorStatus_EXITING {
			continue
		}

		validatorKey := fmt.Sprintf("%#x", bytesutil.Trunc(duty.PublicKey))
		attesterIndex := duty.AttesterSlot - slotOffset
		if attesterIndex >= params.BeaconConfig().SlotsPerEpoch {
			log.WithField("duty", duty).Warn("Invalid attester slot")
		} else {
			attesterKeys[duty.AttesterSlot-slotOffset] = append(attesterKeys[duty.AttesterSlot-slotOffset], validatorKey)
			totalAttestingKeys++
			if v.emitAccountMetrics {
				ValidatorNextAttestationSlotGaugeVec.WithLabelValues(validatorNotTruncatedKey).Set(float64(duty.AttesterSlot))
			}
		}

		for _, proposerSlot := range duty.ProposerSlots {
			proposerIndex := proposerSlot - slotOffset
			if proposerIndex >= params.BeaconConfig().SlotsPerEpoch {
				log.WithField("duty", duty).Warn("Invalid proposer slot")
			} else {
				proposerKeys[proposerIndex] = validatorKey
			}
			if v.emitAccountMetrics {
				ValidatorNextProposalSlotGaugeVec.WithLabelValues(validatorNotTruncatedKey).Set(float64(proposerSlot))
			}
		}
	}
	for i := types.Slot(0); i < params.BeaconConfig().SlotsPerEpoch; i++ {
		startTime := slots.StartTime(v.genesisTime, slotOffset+i)
		durationTillDuty := time.Until(startTime)

		if len(attesterKeys[i]) > 0 {
			log.WithFields(logrus.Fields{
				"slot":                  slotOffset + i,
				"slotInEpoch":           (slotOffset + i) % params.BeaconConfig().SlotsPerEpoch,
				"timeTillDuty":          durationTillDuty.Round(time.Second),
				"attesterDutiesAtSlot":  len(attesterKeys[i]),
				"totalAttestersInEpoch": totalAttestingKeys,
				"pubKeys":               attesterKeys[i],
			}).Info("Attestation schedule")
		}
		if proposerKeys[i] != "" {
			log.WithField("slot", slotOffset+i).WithField("timeTillDuty", durationTillDuty.Round(time.Second)).WithField("pubKey", proposerKeys[i]).Info("Proposal schedule")
		}
	}
}

// PushProposerSettings calls the prepareBeaconProposer RPC to set the fee recipient and also the register validator API if using a custom builder.
func (v *validator) PushProposerSettings(ctx context.Context, km keymanager.IKeymanager) error {
	// only used after Bellatrix
	if v.ProposerSettings == nil {
		e := params.BeaconConfig().BellatrixForkEpoch
		if e != math.MaxUint64 && slots.ToEpoch(slots.CurrentSlot(v.genesisTime)) < e {
			log.Warn("You will need to specify the Ethereum addresses which will receive transaction fee rewards from proposing blocks. " +
				"This is known as a fee recipient configuration. You can read more about this feature in our documentation portal here (https://docs.prylabs.network/docs/execution-node/fee-recipient)")
		} else {
			log.Warn("In order to receive transaction fees from proposing blocks post merge, " +
				"you must specify a configuration known as a fee recipient config. " +
				"If it is not provided, transaction fees will be burnt. Please see our documentation for more information on this requirement (https://docs.prylabs.network/docs/execution-node/fee-recipient).")
		}
		return nil
	}
	if km == nil {
		return errors.New("keymanager is nil when calling PrepareBeaconProposer")
	}

	deadline := v.SlotDeadline(slots.RoundUpToNearestEpoch(slots.CurrentSlot(v.genesisTime)))
	ctx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	pubkeys, err := km.FetchValidatingPublicKeys(ctx)
	if err != nil {
		return err
	}
	if len(pubkeys) == 0 {
		log.Info("No imported public keys. Skipping prepare proposer routine")
		return nil
	}
	proposerReqs, err := v.buildPrepProposerReqs(ctx, pubkeys)
	if err != nil {
		return err
	}
	if len(proposerReqs) == 0 {
		log.Warnf("Could not locate valid validator indices. Skipping prepare proposer routine")
		return nil
	}
	if len(proposerReqs) != len(pubkeys) {
		log.WithFields(logrus.Fields{
			"pubkeysCount": len(pubkeys),
			"reqCount":     len(proposerReqs),
		}).Warnln("Prepare proposer request did not success with all pubkeys")
	}
	if _, err := v.validatorClient.PrepareBeaconProposer(ctx, &ethpb.PrepareBeaconProposerRequest{
		Recipients: proposerReqs,
	}); err != nil {
		return err
	}

	signedRegReqs, err := v.buildSignedRegReqs(ctx, pubkeys, km.Sign)
	if err != nil {
		return err
	}
	if err := SubmitValidatorRegistrations(ctx, v.validatorClient, signedRegReqs); err != nil {
		return errors.Wrap(ErrBuilderValidatorRegistration, err.Error())
	}

	return nil
}

func (v *validator) buildPrepProposerReqs(ctx context.Context, pubkeys [][fieldparams.BLSPubkeyLength]byte) ([]*ethpb.PrepareBeaconProposerRequest_FeeRecipientContainer, error) {
	var prepareProposerReqs []*ethpb.PrepareBeaconProposerRequest_FeeRecipientContainer

	for _, k := range pubkeys {
		validatorIndex, ok := v.pubkeyToValidatorIndex[k]
		// Get validator index from RPC server if not found.
		if !ok {
			i, ok, err := v.validatorIndex(ctx, k)
			if err != nil {
				return nil, err
			}
			if !ok { // Nothing we can do if RPC server doesn't have validator index.
				continue
			}
			validatorIndex = i
			v.pubkeyToValidatorIndex[k] = i
		}
		feeRecipient := common.HexToAddress(params.BeaconConfig().EthBurnAddressHex)
		if v.ProposerSettings.DefaultConfig != nil {
			feeRecipient = v.ProposerSettings.DefaultConfig.FeeRecipient // Use cli config for fee recipient.
		}
		if v.ProposerSettings.ProposeConfig != nil {
			config, ok := v.ProposerSettings.ProposeConfig[k]
			if ok && config != nil {
				feeRecipient = config.FeeRecipient // Use file config for fee recipient.
			}
		}
		prepareProposerReqs = append(prepareProposerReqs, &ethpb.PrepareBeaconProposerRequest_FeeRecipientContainer{
			ValidatorIndex: validatorIndex,
			FeeRecipient:   feeRecipient[:],
		})
		if hexutil.Encode(feeRecipient.Bytes()) == params.BeaconConfig().EthBurnAddressHex {
			log.WithFields(logrus.Fields{
				"validatorIndex": validatorIndex,
				"feeRecipient":   feeRecipient,
			}).Warn("Fee recipient is burn address")
		}
	}
	return prepareProposerReqs, nil
}

func (v *validator) buildSignedRegReqs(ctx context.Context, pubkeys [][fieldparams.BLSPubkeyLength]byte, signer iface.SigningFunc) ([]*ethpb.SignedValidatorRegistrationV1, error) {
	var signedValRegRegs []*ethpb.SignedValidatorRegistrationV1

	for i, k := range pubkeys {
		feeRecipient := common.HexToAddress(params.BeaconConfig().EthBurnAddressHex)
		gasLimit := params.BeaconConfig().DefaultBuilderGasLimit
		enabled := false
		if v.ProposerSettings.DefaultConfig != nil {
			feeRecipient = v.ProposerSettings.DefaultConfig.FeeRecipient // Use cli config for fee recipient.
			config := v.ProposerSettings.DefaultConfig.BuilderConfig
			if config != nil && config.Enabled {
				gasLimit = uint64(config.GasLimit) // Use cli config for gas limit.
				enabled = true
			}
		}
		if v.ProposerSettings.ProposeConfig != nil {
			config, ok := v.ProposerSettings.ProposeConfig[k]
			if ok && config != nil {
				feeRecipient = config.FeeRecipient // Use file config for fee recipient.
				builderConfig := config.BuilderConfig
				if builderConfig != nil {
					if builderConfig.Enabled {
						gasLimit = uint64(builderConfig.GasLimit) // Use file config for gas limit.
						enabled = true
					} else {
						enabled = false // Custom config can disable validator from register.
					}
				}
			}
		}
		if !enabled {
			continue
		}
		req := &ethpb.ValidatorRegistrationV1{
			FeeRecipient: feeRecipient[:],
			GasLimit:     gasLimit,
			Timestamp:    uint64(time.Now().UTC().Unix()),
			Pubkey:       pubkeys[i][:],
		}
		signedReq, err := v.SignValidatorRegistrationRequest(ctx, signer, req)
		if err != nil {
			log.WithFields(logrus.Fields{
				"pubkey":       fmt.Sprintf("%#x", req.Pubkey),
				"feeRecipient": feeRecipient,
			}).Error(err)
			continue
		}
		signedValRegRegs = append(signedValRegRegs, signedReq)
		if hexutil.Encode(feeRecipient.Bytes()) == params.BeaconConfig().EthBurnAddressHex {
			log.WithFields(logrus.Fields{
				"pubkey":       fmt.Sprintf("%#x", req.Pubkey),
				"feeRecipient": feeRecipient,
			}).Warn("Fee recipient is burn address")
		}
	}
	return signedValRegRegs, nil
}

func (v *validator) validatorIndex(ctx context.Context, pubkey [fieldparams.BLSPubkeyLength]byte) (types.ValidatorIndex, bool, error) {
	resp, err := v.validatorClient.ValidatorIndex(ctx, &ethpb.ValidatorIndexRequest{PublicKey: pubkey[:]})
	switch {
	case status.Code(err) == codes.NotFound:
		log.Warnf("Could not find validator index for public key %#x. "+
			"Perhaps the validator is not yet active.", pubkey)
		return 0, false, nil
	case err != nil:
		return 0, false, err
	}
	return resp.Index, true, nil
}

// This constructs a validator subscribed key, it's used to track
// which subnet has already been pending requested.
func validatorSubscribeKey(slot types.Slot, committeeID types.CommitteeIndex) [64]byte {
	return bytesutil.ToBytes64(append(bytesutil.Bytes32(uint64(slot)), bytesutil.Bytes32(uint64(committeeID))...))
}

// This tracks all validators' voting status.
type voteStats struct {
	startEpoch          types.Epoch
	totalAttestedCount  uint64
	totalRequestedCount uint64
	totalDistance       types.Slot
	totalCorrectSource  uint64
	totalCorrectTarget  uint64
	totalCorrectHead    uint64
}

// This tracks all validators' submissions for sync committees.
type syncCommitteeStats struct {
	totalMessagesSubmitted uint64
}
