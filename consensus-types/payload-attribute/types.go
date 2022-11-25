package payloadattribute

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
)

var (
	_ = Attributer(&data{})
)

type data struct {
	version               int
	timeStamp             uint64
	prevRandao            []byte
	suggestedFeeRecipient []byte
	withdrawals           []*enginev1.Withdrawal
}

var (
	errNilPayloadAttribute         = errors.New("received nil payload attribute")
	ErrUnsupportedPayloadAttribute = errors.New("unsupported payload attribute")
)

func New(i interface{}) (Attributer, error) {
	switch a := i.(type) {
	case nil:
		return nil, blocks.ErrNilObject
	case *enginev1.PayloadAttributes:
		return initPayloadAttributeFromV1(a)
	case *enginev1.PayloadAttributesV2:
		return initPayloadAttributeFromV2(a)
	default:
		return nil, errors.Wrapf(ErrUnsupportedPayloadAttribute, "unable to create payload attribute from type %T", i)
	}
}

func initPayloadAttributeFromV1(a *enginev1.PayloadAttributes) (Attributer, error) {
	if a == nil {
		return nil, errNilPayloadAttribute
	}

	return &data{
		version:               version.Capella,
		prevRandao:            a.PrevRandao,
		timeStamp:             a.Timestamp,
		suggestedFeeRecipient: a.SuggestedFeeRecipient,
	}, nil
}

func initPayloadAttributeFromV2(a *enginev1.PayloadAttributesV2) (Attributer, error) {
	if a == nil {
		return nil, errNilPayloadAttribute
	}

	return &data{
		version:               version.Capella,
		prevRandao:            a.PrevRandao,
		timeStamp:             a.Timestamp,
		suggestedFeeRecipient: a.SuggestedFeeRecipient,
		withdrawals:           a.Withdrawals,
	}, nil
}
