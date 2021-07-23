package v2

import (
	statepb "github.com/prysmaticlabs/prysm/proto/prysm/v2/state"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

// CopySyncCommittee copies the provided sync committee object.
func CopySyncCommittee(data *statepb.SyncCommittee) *statepb.SyncCommittee {
	if data == nil {
		return nil
	}
	return &statepb.SyncCommittee{
		Pubkeys:         bytesutil.Copy2dBytes(data.Pubkeys),
		AggregatePubkey: bytesutil.SafeCopyBytes(data.AggregatePubkey),
	}
}
