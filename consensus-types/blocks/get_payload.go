package blocks

import (
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	pb "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	"google.golang.org/protobuf/proto"
)

// GetPayloadResponse represents the result of unmarshaling an execution engine
// GetPayloadResponseV(1|2|3|4) value.
type GetPayloadResponse struct {
	ExecutionData   interfaces.ExecutionData
	BlobsBundle     *enginev1.BlobsBundle
	OverrideBuilder bool
	// todo: should we convert this to Gwei up front?
	Bid primitives.Wei
}

// bundleGetter is an interface satisfied by get payload responses that have a blobs bundle.
type bundleGetter interface {
	GetBlobsBundle() *pb.BlobsBundle
}

// bidValueGetter is an interface satisfied by get payload responses that have a bid value.
type bidValueGetter interface {
	GetValue() []byte
}

type shouldOverrideBuilderGetter interface {
	GetShouldOverrideBuilder() bool
}

func NewGetPayloadResponse(msg proto.Message) (*GetPayloadResponse, error) {
	r := &GetPayloadResponse{}
	bundleGetter, hasBundle := msg.(bundleGetter)
	if hasBundle {
		r.BlobsBundle = bundleGetter.GetBlobsBundle()
	}
	bidValueGetter, hasBid := msg.(bidValueGetter)
	wei := primitives.ZeroWei
	if hasBid {
		wei = primitives.LittleEndianBytesToWei(bidValueGetter.GetValue())
		r.Bid = wei
	}
	shouldOverride, hasShouldOverride := msg.(shouldOverrideBuilderGetter)
	if hasShouldOverride {
		r.OverrideBuilder = shouldOverride.GetShouldOverrideBuilder()
	}
	ed, err := NewWrappedExecutionData(msg, wei)
	if err != nil {
		return nil, err
	}
	r.ExecutionData = ed
	return r, nil
}
