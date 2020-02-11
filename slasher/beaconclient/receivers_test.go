package beaconclient

import (
	"testing"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"

	"github.com/prysmaticlabs/prysm/shared/mock"
)

func TestService_ReceiveBlocks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconChainClient(ctrl)

	bs := Service{
		client: client,
	}
	stream := mock.NewMockBeaconChain_StreamBlocksClient(ctrl)
	client.EXPECT().StreamBlocks(
		gomock.Any(),
		&ptypes.Empty{},
	).Return(stream, nil)
	stream.EXPECT().Recv().Return(
		&ethpb.ChainStartResponse{
			Started:     true,
			GenesisTime: genesis,
		},
		nil,
	)
	if err := v.WaitForChainStart(context.Background()); err != nil {
		t.Fatal(err)
	}
	if v.genesisTime != genesis {
		t.Errorf("Expected chain start time to equal %d, received %d", genesis, v.genesisTime)
	}
	if v.ticker == nil {
		t.Error("Expected ticker to be set, received nil")
	}
}
