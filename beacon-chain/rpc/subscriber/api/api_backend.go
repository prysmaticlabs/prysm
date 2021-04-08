package api

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/event"
	eth2Types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/beacon"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/subscriber/api/events"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/params"
	log "github.com/sirupsen/logrus"
	"time"
)

type APIBackend struct {
	BeaconChain       beacon.Server
	consensusInfoFeed event.Feed
	scope             event.SubscriptionScope
}

func (backend *APIBackend) SubscribeNewEpochEvent(
	ctx context.Context,
	epoch eth2Types.Epoch,
	consensusChannel chan interface{},
) {
	beaconChain := backend.BeaconChain
	headFetcher := beaconChain.HeadFetcher
	headState, err := headFetcher.HeadState(ctx)

	if err != nil {
		log.Warn("Could not access head state")

		return
	}

	if headState == nil {
		err := fmt.Errorf("we are not ready to serve information")
		log.WithField("fromEpoch", epoch).Error(err.Error())

		return
	}

	go handleMinimalConsensusSubscription(
		ctx,
		headFetcher,
		headState,
		beaconChain,
		consensusChannel,
	)

	return
}

func (backend *APIBackend) GetMinimalConsensusInfo(ctx context.Context, epoch eth2Types.Epoch) (*events.MinimalEpochConsensusInfo, error) {
	return backend.BeaconChain.GetMinimalConsensusInfo(ctx, epoch)
}

func (backend *APIBackend) GetMinimalConsensusInfoRange(
	ctx context.Context,
	epoch eth2Types.Epoch,
) ([]*events.MinimalEpochConsensusInfo, error) {
	return backend.BeaconChain.GetMinimalConsensusInfoRange(ctx, epoch)
}

func (backend *APIBackend) FutureMinimalConsensusInfo(ctx context.Context) (*events.MinimalEpochConsensusInfo, error) {
	return backend.BeaconChain.FutureEpochProposerList(ctx)
}

func handleMinimalConsensusSubscription(
	ctx context.Context,
	headFetcher blockchain.HeadFetcher,
	headState *state.BeaconState,
	beaconChain beacon.Server,
	consensusInfoChannel chan interface{},
) {
	subscriptionStartEpoch := eth2Types.Epoch(headState.Slot()/params.BeaconConfig().SlotsPerEpoch - 1)
	stateChannel := make(chan *feed.Event, params.BeaconConfig().SlotsPerEpoch)
	stateNotifier := beaconChain.StateNotifier
	stateFeed := stateNotifier.StateFeed()
	stateSubscription := stateFeed.Subscribe(stateChannel)
	lastSentEpoch := subscriptionStartEpoch

	defer stateSubscription.Unsubscribe()

	if nil == headState {
		panic("head state cannot be nil")
	}

	log.WithField("fromEpoch", subscriptionStartEpoch).Info("registered new subscriber for consensus info")

	sendConsensusInfo := func()(err error) {
		consensusInfo, currentErr := beaconChain.FutureEpochProposerList(beaconChain.Ctx)

		if nil != currentErr {
			log.WithField("currentEpoch", consensusInfo).WithField("err", currentErr).
				Error("could not retrieve epoch in subscription")
		}

		if nil == consensusInfo {
			return
		}

		blockEpoch := eth2Types.Epoch(consensusInfo.Epoch)

		// Epoch did not progress
		if blockEpoch == lastSentEpoch {
			return
		}

		consensusInfoChannel <- consensusInfo
		lastSentEpoch = eth2Types.Epoch(consensusInfo.Epoch)

		return
	}

	tickerDuration := time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Second
	ticker := time.NewTicker(tickerDuration)

	for {
		select {
		case stateEvent := <-stateChannel:
			if statefeed.BlockProcessed != stateEvent.Type {
				continue
			}

			ticker.Reset(tickerDuration)
			currentErr := sendConsensusInfo()

			if nil != currentErr {
				log.WithField("err", currentErr).Error("could not fetch state during minimalConsensusInfoCheck")
			}
		case <-ticker.C:
			currentErr := sendConsensusInfo()

			if nil != currentErr {
				log.WithField("err", currentErr).Error("could not fetch state during minimalConsensusInfoCheck")
			}
		}
	}
}
