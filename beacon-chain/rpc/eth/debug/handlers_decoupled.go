package debug

import (
	"fmt"
	"strconv"

	"github.com/OffchainLabs/prysm/v7/api/server/structs"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/forkchoice/decoupled"
	"github.com/OffchainLabs/prysm/v7/time/slots"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// Standard dump shape, so existing consumers keep parsing.
func decoupledForkChoiceResponse(d *decoupled.Dump) *structs.DecoupledForkChoiceDumpResponse {
	nodes := make([]*structs.DecoupledForkChoiceNode, len(d.Nodes))
	for i, n := range d.Nodes {
		nodes[i] = &structs.DecoupledForkChoiceNode{
			Slot:               fmt.Sprintf("%d", n.Slot),
			BlockRoot:          hexutil.Encode(n.Root[:]),
			ParentRoot:         hexutil.Encode(n.Parent[:]),
			JustifiedEpoch:     "0",
			FinalizedEpoch:     "0",
			Weight:             fmt.Sprintf("%d", n.Votes),
			Validity:           "valid",
			ExecutionBlockHash: hexutil.Encode(make([]byte, 32)),
			ExtraData: &structs.DecoupledForkChoiceNodeExtraData{
				TargetHeight:    fmt.Sprintf("%d", n.FG.Target.Height),
				TargetRoot:      hexutil.Encode(n.FG.Target.Root[:]),
				JustifiedHeight: fmt.Sprintf("%d", n.FG.Justified.Height),
				JustifiedRoot:   hexutil.Encode(n.FG.Justified.Root[:]),
				FinalizedHeight: fmt.Sprintf("%d", n.FG.Finalized.Height),
				FinalizedRoot:   hexutil.Encode(n.FG.Finalized.Root[:]),
				Votes:           fmt.Sprintf("%d", n.Votes),
				SupportVotes:    fmt.Sprintf("%d", n.Support),
				Viable:          n.Viable,
				Confirmed:       n.Confirmed,
				Locked:          n.Locked,
				Processed:       strconv.FormatInt(n.Processed.Unix(), 10),
			},
		}
	}
	locks := make([]*structs.DecoupledForkChoiceLock, len(d.Locks))
	for i, l := range d.Locks {
		locks[i] = &structs.DecoupledForkChoiceLock{
			Root:     hexutil.Encode(l.Root[:]),
			Slot:     fmt.Sprintf("%d", l.Slot),
			Deadline: strconv.FormatInt(l.Deadline.Unix(), 10),
		}
	}
	fast := make([]string, len(d.FastConfirmed))
	for i, r := range d.FastConfirmed {
		fast[i] = hexutil.Encode(r[:])
	}
	saved := make([]*structs.DecoupledForkChoiceSaved, len(d.Saved))
	for i, s := range d.Saved {
		saved[i] = &structs.DecoupledForkChoiceSaved{Slot: fmt.Sprintf("%d", s.Slot), Root: hexutil.Encode(s.Root[:])}
	}
	voters := make([]*structs.DecoupledForkChoiceVoters, len(d.Voters))
	for i, v := range d.Voters {
		voters[i] = &structs.DecoupledForkChoiceVoters{Slot: fmt.Sprintf("%d", v.Slot), Count: fmt.Sprintf("%d", v.Count)}
	}
	return &structs.DecoupledForkChoiceDumpResponse{
		JustifiedCheckpoint: &structs.Checkpoint{Epoch: fmt.Sprintf("%d", slots.ToEpoch(d.JustifiedSlot)), Root: hexutil.Encode(d.Justified[:])},
		FinalizedCheckpoint: &structs.Checkpoint{Epoch: fmt.Sprintf("%d", slots.ToEpoch(d.FinalizedSlot)), Root: hexutil.Encode(d.Finalized[:])},
		ForkChoiceNodes:     nodes,
		ExtraData: &structs.DecoupledForkChoiceDumpExtraData{
			Decoupled:           true,
			Slot:                fmt.Sprintf("%d", d.Slot),
			Joined:              d.Joined,
			RestartTime:         strconv.FormatInt(d.RestartTime.Unix(), 10),
			HeadRoot:            hexutil.Encode(d.Head[:]),
			ConfirmedRoot:       hexutil.Encode(d.Confirmed[:]),
			LatestConfirmedRoot: hexutil.Encode(d.LatestConfirmed[:]),
			JustifiedHeight:     fmt.Sprintf("%d", d.JustifiedHeight),
			JustifiedSlot:       fmt.Sprintf("%d", d.JustifiedSlot),
			FinalizedHeight:     fmt.Sprintf("%d", d.FinalizedHeight),
			FinalizedSlot:       fmt.Sprintf("%d", d.FinalizedSlot),
			MaxHeight:           fmt.Sprintf("%d", d.MaxHeight),
			FrameRoot:           hexutil.Encode(d.Frame[:]),
			Floor:               fmt.Sprintf("%d", d.Floor),
			CommitteeSize:       fmt.Sprintf("%d", d.CommitteeSize),
			Locks:               locks,
			FastConfirmed:       fast,
			Saved:               saved,
			Voters:              voters,
		},
	}
}
