package lightclient

import (
	"testing"

	lightclient "github.com/prysmaticlabs/prysm/v5/beacon-chain/core/light-client"

	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	ethpbv1 "github.com/prysmaticlabs/prysm/v5/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v5/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
)

// When the update has relevant sync committee
func createNonEmptySyncCommitteeBranch() [][]byte {
	res := make([][]byte, fieldparams.SyncCommitteeBranchDepth)
	res[0] = []byte("xyz")
	return res
}

// When the update has finality
func createNonEmptyFinalityBranch() [][]byte {
	res := make([][]byte, lightclient.FinalityBranchNumOfLeaves)
	res[0] = []byte("xyz")
	return res
}

func TestIsBetterUpdate(t *testing.T) {
	testCases := []struct {
		name           string
		oldUpdate      *ethpbv2.LightClientUpdate
		newUpdate      *ethpbv2.LightClientUpdate
		expectedResult bool
	}{
		{
			name: "new has supermajority but old doesn't",
			oldUpdate: &ethpbv2.LightClientUpdate{
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits: []byte{0b01111100, 0b1}, // [0,0,1,1,1,1,1,0]
				},
			},
			newUpdate: &ethpbv2.LightClientUpdate{
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits: []byte{0b11111100, 0b1}, // [0,0,1,1,1,1,1,1]
				},
			},
			expectedResult: true,
		},
		{
			name: "old has supermajority but new doesn't",
			oldUpdate: &ethpbv2.LightClientUpdate{
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits: []byte{0b11111100, 0b1}, // [0,0,1,1,1,1,1,1]
				},
			},
			newUpdate: &ethpbv2.LightClientUpdate{
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits: []byte{0b01111100, 0b1}, // [0,0,1,1,1,1,1,0]
				},
			},
			expectedResult: false,
		},
		{
			name: "new doesn't have supermajority and newNumActiveParticipants is greater than oldNumActiveParticipants",
			oldUpdate: &ethpbv2.LightClientUpdate{
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits: []byte{0b00111100, 0b1}, // [0,0,1,1,1,1,0,0]
				},
			},
			newUpdate: &ethpbv2.LightClientUpdate{
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits: []byte{0b01111100, 0b1}, // [0,0,1,1,1,1,1,0]
				},
			},
			expectedResult: true,
		},
		{
			name: "new doesn't have supermajority and newNumActiveParticipants is lesser than oldNumActiveParticipants",
			oldUpdate: &ethpbv2.LightClientUpdate{
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits: []byte{0b01111100, 0b1}, // [0,0,1,1,1,1,1,0]
				},
			},
			newUpdate: &ethpbv2.LightClientUpdate{
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits: []byte{0b00111100, 0b1}, // [0,0,1,1,1,1,0,0]
				},
			},
			expectedResult: false,
		},
		{
			name: "new has relevant sync committee but old doesn't",
			oldUpdate: &ethpbv2.LightClientUpdate{
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits: []byte{0b00111100, 0b1}, // [0,0,1,1,1,1,0,0]
				},
				AttestedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 1000000,
						}},
					},
				},
				NextSyncCommitteeBranch: make([][]byte, fieldparams.SyncCommitteeBranchDepth),
				SignatureSlot:           9999,
			},
			newUpdate: &ethpbv2.LightClientUpdate{
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits: []byte{0b00111100, 0b1}, // [0,0,1,1,1,1,0,0]
				},
				AttestedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 1000001,
						}},
					},
				},
				NextSyncCommitteeBranch: createNonEmptySyncCommitteeBranch(),
				SignatureSlot:           1000000,
			},
			expectedResult: true,
		},
		{
			name: "old has relevant sync committee but new doesn't",
			oldUpdate: &ethpbv2.LightClientUpdate{
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits: []byte{0b00111100, 0b1}, // [0,0,1,1,1,1,0,0]
				},
				AttestedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 1000001,
						}},
					},
				},
				NextSyncCommitteeBranch: createNonEmptySyncCommitteeBranch(),
				SignatureSlot:           1000000,
			},
			newUpdate: &ethpbv2.LightClientUpdate{
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits: []byte{0b00111100, 0b1}, // [0,0,1,1,1,1,0,0]
				},
				AttestedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 1000000,
						}},
					},
				},
				NextSyncCommitteeBranch: make([][]byte, fieldparams.SyncCommitteeBranchDepth),
				SignatureSlot:           9999,
			},
			expectedResult: false,
		},
		{
			name: "new has finality but old doesn't",
			oldUpdate: &ethpbv2.LightClientUpdate{
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits: []byte{0b00111100, 0b1}, // [0,0,1,1,1,1,0,0]
				},
				AttestedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 1000000,
						}},
					},
				},
				NextSyncCommitteeBranch: createNonEmptySyncCommitteeBranch(),
				SignatureSlot:           9999,
				FinalityBranch:          make([][]byte, lightclient.FinalityBranchNumOfLeaves),
			},
			newUpdate: &ethpbv2.LightClientUpdate{
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits: []byte{0b00111100, 0b1}, // [0,0,1,1,1,1,0,0]
				},
				AttestedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 1000000,
						}},
					},
				},
				NextSyncCommitteeBranch: createNonEmptySyncCommitteeBranch(),
				SignatureSlot:           9999,
				FinalityBranch:          createNonEmptyFinalityBranch(),
			},
			expectedResult: true,
		},
		{
			name: "old has finality but new doesn't",
			oldUpdate: &ethpbv2.LightClientUpdate{
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits: []byte{0b00111100, 0b1}, // [0,0,1,1,1,1,0,0]
				},
				AttestedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 1000000,
						}},
					},
				},
				NextSyncCommitteeBranch: createNonEmptySyncCommitteeBranch(),
				SignatureSlot:           9999,
				FinalityBranch:          createNonEmptyFinalityBranch(),
			},
			newUpdate: &ethpbv2.LightClientUpdate{
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits: []byte{0b00111100, 0b1}, // [0,0,1,1,1,1,0,0]
				},
				AttestedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 1000000,
						}},
					},
				},
				NextSyncCommitteeBranch: createNonEmptySyncCommitteeBranch(),
				SignatureSlot:           9999,
				FinalityBranch:          make([][]byte, lightclient.FinalityBranchNumOfLeaves),
			},
			expectedResult: false,
		},
		{
			name: "new has finality and sync committee finality both but old doesn't have sync committee finality",
			oldUpdate: &ethpbv2.LightClientUpdate{
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits: []byte{0b00111100, 0b1}, // [0,0,1,1,1,1,0,0]
				},
				AttestedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 1000000,
						}},
					},
				},
				NextSyncCommitteeBranch: createNonEmptySyncCommitteeBranch(),
				SignatureSlot:           9999,
				FinalityBranch:          createNonEmptyFinalityBranch(),
				FinalizedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 9999,
						}},
					},
				},
			},
			newUpdate: &ethpbv2.LightClientUpdate{
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits: []byte{0b00111100, 0b1}, // [0,0,1,1,1,1,0,0]
				},
				AttestedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 1000000,
						}},
					},
				},
				NextSyncCommitteeBranch: createNonEmptySyncCommitteeBranch(),
				SignatureSlot:           999999,
				FinalityBranch:          createNonEmptyFinalityBranch(),
				FinalizedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 999999,
						}},
					},
				},
			},
			expectedResult: true,
		},
		{
			name: "new has finality but doesn't have sync committee finality and old has sync committee finality",
			oldUpdate: &ethpbv2.LightClientUpdate{
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits: []byte{0b00111100, 0b1}, // [0,0,1,1,1,1,0,0]
				},
				AttestedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 1000000,
						}},
					},
				},
				NextSyncCommitteeBranch: createNonEmptySyncCommitteeBranch(),
				SignatureSlot:           999999,
				FinalityBranch:          createNonEmptyFinalityBranch(),
				FinalizedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 999999,
						}},
					},
				},
			},
			newUpdate: &ethpbv2.LightClientUpdate{
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits: []byte{0b00111100, 0b1}, // [0,0,1,1,1,1,0,0]
				},
				AttestedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 1000000,
						}},
					},
				},
				NextSyncCommitteeBranch: createNonEmptySyncCommitteeBranch(),
				SignatureSlot:           9999,
				FinalityBranch:          createNonEmptyFinalityBranch(),
				FinalizedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 9999,
						}},
					},
				},
			},
			expectedResult: false,
		},
		{
			name: "new has more active participants than old",
			oldUpdate: &ethpbv2.LightClientUpdate{
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits: []byte{0b00111100, 0b1}, // [0,0,1,1,1,1,0,0]
				},
				AttestedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 1000000,
						}},
					},
				},
				NextSyncCommitteeBranch: createNonEmptySyncCommitteeBranch(),
				SignatureSlot:           9999,
				FinalityBranch:          createNonEmptyFinalityBranch(),
				FinalizedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 9999,
						}},
					},
				},
			},
			newUpdate: &ethpbv2.LightClientUpdate{
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits: []byte{0b01111100, 0b1}, // [0,1,1,1,1,1,0,0]
				},
				AttestedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 1000000,
						}},
					},
				},
				NextSyncCommitteeBranch: createNonEmptySyncCommitteeBranch(),
				SignatureSlot:           9999,
				FinalityBranch:          createNonEmptyFinalityBranch(),
				FinalizedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 9999,
						}},
					},
				},
			},
			expectedResult: true,
		},
		{
			name: "new has less active participants than old",
			oldUpdate: &ethpbv2.LightClientUpdate{
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits: []byte{0b01111100, 0b1}, // [0,1,1,1,1,1,0,0]
				},
				AttestedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 1000000,
						}},
					},
				},
				NextSyncCommitteeBranch: createNonEmptySyncCommitteeBranch(),
				SignatureSlot:           9999,
				FinalityBranch:          createNonEmptyFinalityBranch(),
				FinalizedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 9999,
						}},
					},
				},
			},
			newUpdate: &ethpbv2.LightClientUpdate{
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits: []byte{0b00111100, 0b1}, // [0,0,1,1,1,1,0,0]
				},
				AttestedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 1000000,
						}},
					},
				},
				NextSyncCommitteeBranch: createNonEmptySyncCommitteeBranch(),
				SignatureSlot:           9999,
				FinalityBranch:          createNonEmptyFinalityBranch(),
				FinalizedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 9999,
						}},
					},
				},
			},
			expectedResult: false,
		},
		{
			name: "new's attested header's slot is lesser than old's attested header's slot",
			oldUpdate: &ethpbv2.LightClientUpdate{
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits: []byte{0b00111100, 0b1}, // [0,0,1,1,1,1,0,0]
				},
				AttestedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 1000000,
						}},
					},
				},
				NextSyncCommitteeBranch: createNonEmptySyncCommitteeBranch(),
				SignatureSlot:           9999,
				FinalityBranch:          createNonEmptyFinalityBranch(),
				FinalizedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 9999,
						}},
					},
				},
			},
			newUpdate: &ethpbv2.LightClientUpdate{
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits: []byte{0b00111100, 0b1}, // [0,0,1,1,1,1,0,0]
				},
				AttestedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 999999,
						}},
					},
				},
				NextSyncCommitteeBranch: createNonEmptySyncCommitteeBranch(),
				SignatureSlot:           9999,
				FinalityBranch:          createNonEmptyFinalityBranch(),
				FinalizedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 9999,
						}},
					},
				},
			},
			expectedResult: true,
		},
		{
			name: "new's attested header's slot is greater than old's attested header's slot",
			oldUpdate: &ethpbv2.LightClientUpdate{
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits: []byte{0b00111100, 0b1}, // [0,0,1,1,1,1,0,0]
				},
				AttestedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 999999,
						}},
					},
				},
				NextSyncCommitteeBranch: createNonEmptySyncCommitteeBranch(),
				SignatureSlot:           9999,
				FinalityBranch:          createNonEmptyFinalityBranch(),
				FinalizedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 9999,
						}},
					},
				},
			},
			newUpdate: &ethpbv2.LightClientUpdate{
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits: []byte{0b00111100, 0b1}, // [0,0,1,1,1,1,0,0]
				},
				AttestedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 1000000,
						}},
					},
				},
				NextSyncCommitteeBranch: createNonEmptySyncCommitteeBranch(),
				SignatureSlot:           9999,
				FinalityBranch:          createNonEmptyFinalityBranch(),
				FinalizedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 9999,
						}},
					},
				},
			},
			expectedResult: false,
		},
		{
			name: "none of the above conditions are met and new signature's slot is less than old signature's slot",
			oldUpdate: &ethpbv2.LightClientUpdate{
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits: []byte{0b00111100, 0b1}, // [0,0,1,1,1,1,0,0]
				},
				AttestedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 1000000,
						}},
					},
				},
				NextSyncCommitteeBranch: createNonEmptySyncCommitteeBranch(),
				SignatureSlot:           9999,
				FinalityBranch:          createNonEmptyFinalityBranch(),
				FinalizedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 9999,
						}},
					},
				},
			},
			newUpdate: &ethpbv2.LightClientUpdate{
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits: []byte{0b00111100, 0b1}, // [0,0,1,1,1,1,0,0]
				},
				AttestedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 1000000,
						}},
					},
				},
				NextSyncCommitteeBranch: createNonEmptySyncCommitteeBranch(),
				SignatureSlot:           9998,
				FinalityBranch:          createNonEmptyFinalityBranch(),
				FinalizedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 9999,
						}},
					},
				},
			},
			expectedResult: true,
		},
		{
			name: "none of the above conditions are met and new signature's slot is greater than old signature's slot",
			oldUpdate: &ethpbv2.LightClientUpdate{
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits: []byte{0b00111100, 0b1}, // [0,0,1,1,1,1,0,0]
				},
				AttestedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 1000000,
						}},
					},
				},
				NextSyncCommitteeBranch: createNonEmptySyncCommitteeBranch(),
				SignatureSlot:           9998,
				FinalityBranch:          createNonEmptyFinalityBranch(),
				FinalizedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 9999,
						}},
					},
				},
			},
			newUpdate: &ethpbv2.LightClientUpdate{
				SyncAggregate: &ethpbv1.SyncAggregate{
					SyncCommitteeBits: []byte{0b00111100, 0b1}, // [0,0,1,1,1,1,0,0]
				},
				AttestedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 1000000,
						}},
					},
				},
				NextSyncCommitteeBranch: createNonEmptySyncCommitteeBranch(),
				SignatureSlot:           9999,
				FinalityBranch:          createNonEmptyFinalityBranch(),
				FinalizedHeader: &ethpbv2.LightClientHeaderContainer{
					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
						HeaderAltair: &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
							Slot: 9999,
						}},
					},
				},
			},
			expectedResult: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := IsBetterUpdate(testCase.newUpdate, testCase.oldUpdate)
			assert.NoError(t, err)
			assert.Equal(t, testCase.expectedResult, result)
		})
	}
}
