package debug

import (
	"context"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/prysmaticlabs/prysm/shared/params"
	log "github.com/sirupsen/logrus"
)

func (ds *Server) SlashMyProposer(
	ctx context.Context,
	_ *ptypes.Empty,
) (*ptypes.Empty, error) {
	c := params.BeaconConfig()
	c.SlashMyProposerCount++
	params.OverrideBeaconConfig(c)

	log.WithField("count", params.BeaconConfig().SlashMyProposerCount).Info("Slash my proposer 🙊")
	return &ptypes.Empty{}, nil
}
