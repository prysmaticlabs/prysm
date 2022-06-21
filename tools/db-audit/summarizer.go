package main

import "github.com/prysmaticlabs/prysm/beacon-chain/state"

type summarizer struct {
	s state.BeaconState
	sb []byte
	sr SlotRoot
}

func (s *summarizer) Summary() SizeSummary {
	return SizeSummary{
		Total:                         len(s.sb),
		SlotRoot: s.sr,
		GenesisTime:                   s.GenesisTime(),
		GenesisValidatorsRoot:         s.GenesisValidatorsRoot(),
		Slot:                          s.Slot(),
		Fork:                          s.Fork(),
		LatestBlockHeader:             s.LatestBlockHeader(),
		BlockRoots:                    s.BlockRoots(),
		StateRoots:                    s.StateRoots(),
		HistoricalRoots:               s.HistoricalRoots(),
		Eth1Data:                      s.Eth1Data(),
		Eth1DataVotes:                 s.Eth1DataVotes(),
		Eth1DepositIndex:              s.Eth1DepositIndex(),
		Validators:                    s.Validators(),
		Balances:                      s.Balances(),
		RandaoMixes:                   s.RandaoMixes(),
		Slashings:                     s.Slashings(),
		PreviousEpochAttestations:     s.PreviousEpochAttestations(),
		CurrentEpochAttestations:      s.CurrentEpochAttestations(),
		PreviousEpochParticipation:    s.PreviousEpochParticipation(),
		CurrentEpochParticipation:     s.CurrentEpochParticipation(),
		JustificationBits:             s.JustificationBits(),
		PreviouslyJustifiedCheckpoint: s.PreviouslyJustifiedCheckpoint(),
		CurrentJustifiedCheckpoint:    s.CurrentJustifiedCheckpoint(),
		FinalizedCheckpoint:           s.FinalizedCheckpoint(),
		InactivityScores:              s.InactivityScores(),
		CurrentSyncCommittee:          s.CurrentSyncCommittee(),
		NextSyncCommittee:             s.NextSyncCommittee(),
		LatestExecutionPayloadHeader:  s.LatestExecutionPayloadHeader(),
	}
}

func (z *summarizer) GenesisTime() int {
	return 8
}

func (z *summarizer) GenesisValidatorsRoot() int {
	return 32
}

func (z *summarizer) Slot() int {
	return 8
}

func (z *summarizer) Fork() int {
	return z.s.Fork().SizeSSZ()
}

func (z *summarizer) LatestBlockHeader() int {
	return z.s.LatestBlockHeader().SizeSSZ()
}

func (z *summarizer) BlockRoots() int {
	return 32 * len(z.s.BlockRoots())
}

func (z *summarizer) StateRoots() int {
	return 32 * len(z.s.StateRoots())
}

func (z *summarizer) HistoricalRoots() int {
	return 32 * len(z.s.HistoricalRoots())
}

func (z *summarizer) Eth1Data() int {
	return z.s.Eth1Data().SizeSSZ()
}

func (z *summarizer) Eth1DataVotes() int {
	sz := 0
	e1dv := z.s.Eth1DataVotes()
	for _, v := range e1dv {
		sz += v.SizeSSZ()
	}
	return sz
}

func (z *summarizer) Eth1DepositIndex() int {
	return 8
}

func (z *summarizer) Validators() int {
	// Validators is already compressed using integer ids for hashed values
	//return 8 * len(z.s.Validators())
	// JK - not they aren't!!
	sz := 0
	for _, v := range z.s.Validators() {
		sz += v.SizeSSZ()
	}
	return sz
}

func (z *summarizer) Balances() int {
	return 8 * len(z.s.Balances())
}

func (z *summarizer) RandaoMixes() int {
	return 65536 * 32
}

func (z *summarizer) Slashings() int {
	return 8 * len(z.s.Slashings())
}

func (z *summarizer) PreviousEpochAttestations() int {
	atts, err := z.s.PreviousEpochAttestations()
	// just means the state doesn't have this value
	if err != nil {
		return 0
	}
	sz := 0
	for _, v := range atts {
		sz += v.SizeSSZ()
	}
	return sz
}

func (z *summarizer) CurrentEpochAttestations() int {
	atts, err := z.s.CurrentEpochAttestations()
	// just means the state doesn't have this value
	if err != nil {
		return 0
	}
	sz := 0
	for _, v := range atts {
		sz += v.SizeSSZ()
	}
	return sz
}

func (z *summarizer) PreviousEpochParticipation() int {
	p, _ := z.s.PreviousEpochParticipation()
	return len(p)
}

func (z *summarizer) CurrentEpochParticipation() int {
	p, _ := z.s.CurrentEpochParticipation()
	return len(p)
}

func (z *summarizer) JustificationBits() int {
	return len(z.s.JustificationBits())
}

func (z *summarizer) PreviouslyJustifiedCheckpoint() int {
	return z.s.PreviousJustifiedCheckpoint().SizeSSZ()
}

func (z *summarizer) CurrentJustifiedCheckpoint() int {
	return z.s.CurrentJustifiedCheckpoint().SizeSSZ()
}

func (z *summarizer) FinalizedCheckpoint() int {
	return z.s.FinalizedCheckpoint().SizeSSZ()
}

func (z *summarizer) InactivityScores() int {
	scores, err := z.s.InactivityScores()
	// not supported in the fork for the given state
	if err != nil {
		return 0
	}
	return 8 * len(scores)
}

func (z *summarizer) CurrentSyncCommittee() int {
	c, err := z.s.CurrentSyncCommittee()
	if err != nil {
		return 0
	}
	return c.SizeSSZ()
}

func (z *summarizer) NextSyncCommittee() int {
	c, err := z.s.NextSyncCommittee()
	if err != nil {
		return 0
	}
	return c.SizeSSZ()
}

func (z *summarizer) LatestExecutionPayloadHeader() int {
	h, err := z.s.LatestExecutionPayloadHeader()
	if err != nil {
		return 0
	}
	return h.SizeSSZ()
}

/*
func computeSizes(bs state.BeaconState, sb []byte) (*summarizer, error) {
	vu, err := detect.FromState(sb)
	if err != nil {
		return nil, err
	}
	forkName := version.String(vu.Fork)
	switch vu.Fork {
	case version.Phase0:
		st := &ethpb.BeaconState{}
		if err := st.UnmarshalSSZ(sb); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal state, detected fork=%s", forkName)
		}
		return &summarizer{s: bs}, nil
	case version.Altair:
		st := &ethpb.BeaconStateAltair{}
		if err := st.UnmarshalSSZ(sb); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal state, detected fork=%s", forkName)
		}
		return &summarizer{s: bs}, nil
	case version.Bellatrix:
		st := &ethpb.BeaconStateBellatrix{}
		if err := st.UnmarshalSSZ(sb); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal state, detected fork=%s", forkName)
		}
		return &summarizer{s: bs}, nil
	default:
		return nil, fmt.Errorf("unable to initialize BeaconState for fork version=%s", forkName)
	}
}
*/
