package stateutil

import (
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

func HistoricalSummariesRoot(summaries []*ethpb.HistoricalSummary) ([32]byte, error) {
	return SliceRoot(summaries, fieldparams.HistoricalRootsLength)
}
