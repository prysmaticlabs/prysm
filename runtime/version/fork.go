package version

type ForkVersion int

const (
	Phase0 ForkVersion = iota
	Altair
	Bellatrix
	BellatrixBlind
)

func (v ForkVersion) String() string {
	switch v {
	case Phase0:
		return "phase0"
	case Altair:
		return "altair"
	case Bellatrix:
		return "bellatrix"
	case BellatrixBlind:
		return "bellatrix-blind"
	default:
		return "unknown version"
	}
}

func (v ForkVersion) IsPhase0Compatible() bool {
	return v == Phase0
}

func (v ForkVersion) IsAltairCompatible() bool {
	return v == Altair
}

func (v ForkVersion) IsHigherOrEqualToAltair() bool {
	return v >= Altair
}

func (v ForkVersion) IsPreBellatrix() bool {
	return v < Bellatrix
}

func (v ForkVersion) IsBellatrixCompatible() bool {
	return v == Bellatrix || v == BellatrixBlind
}

func (v ForkVersion) IsSyncCommitteeCompatible() bool {
	return v == Altair || v == Bellatrix || v == BellatrixBlind
}

func (v ForkVersion) IsParticipationBitsCompatible() bool {
	return v == Altair || v == Bellatrix || v == BellatrixBlind
}

func (v ForkVersion) IsExecutionPayloadCompatible() bool {
	return v.IsBellatrixCompatible()
}

func (v ForkVersion) IsBlindedBlockCompatible() bool {
	return v == BellatrixBlind
}
