package helpers

import (
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func init() {
	featureconfig.InitFeatureConfig(&featureconfig.FeatureFlagConfig{
		CacheTreeHash: false,
	})
}

func TestEncodeDecodeDepositInput_Ok(t *testing.T) {
	input := &pb.DepositInput{
		Pubkey:                      []byte("key"),
		WithdrawalCredentialsHash32: []byte("withdraw"),
		ProofOfPossession:           []byte("pop"),
	}
	depositTime := time.Now().Unix()
	enc, err := EncodeDepositData(input, params.BeaconConfig().MaxDepositAmount, depositTime)
	if err != nil {
		t.Errorf("Could not encode deposit input: %v", err)
	}
	dec, err := DecodeDepositInput(enc)
	if err != nil {
		t.Errorf("Could not decode deposit input: %v", err)
	}
	if !proto.Equal(input, dec) {
		t.Errorf("Original and decoded messages do not match, wanted %v, received %v", input, dec)
	}
	value, timestamp, err := DecodeDepositAmountAndTimeStamp(enc)
	if err != nil {
		t.Errorf("Could not decode amount and timestamp: %v", err)
	}
	if value != params.BeaconConfig().MaxDepositAmount || timestamp != depositTime {
		t.Errorf(
			"Expected value to match, received %d == %d, expected timestamp to match received %d == %d",
			value,
			params.BeaconConfig().MaxDepositAmount,
			timestamp,
			depositTime,
		)
	}
}

func TestDecodeDepositAmountAndTimeStamp(t *testing.T) {

	tests := []struct {
		depositData *pb.DepositInput
		amount      uint64
		timestamp   int64
	}{
		{
			depositData: &pb.DepositInput{
				Pubkey:                      []byte("testing"),
				ProofOfPossession:           []byte("pop"),
				WithdrawalCredentialsHash32: []byte("withdraw"),
			},
			amount:    8749343850,
			timestamp: 458739850,
		},
		{
			depositData: &pb.DepositInput{
				Pubkey:                      []byte("testing"),
				ProofOfPossession:           []byte("pop"),
				WithdrawalCredentialsHash32: []byte("withdraw"),
			},
			amount:    657660,
			timestamp: 67750,
		},
		{
			depositData: &pb.DepositInput{
				Pubkey:                      []byte("testing"),
				ProofOfPossession:           []byte("pop"),
				WithdrawalCredentialsHash32: []byte("withdraw"),
			},
			amount:    5445540,
			timestamp: 34340,
		}, {
			depositData: &pb.DepositInput{
				Pubkey:                      []byte("testing"),
				ProofOfPossession:           []byte("pop"),
				WithdrawalCredentialsHash32: []byte("withdraw"),
			},
			amount:    4545,
			timestamp: 4343,
		}, {
			depositData: &pb.DepositInput{
				Pubkey:                      []byte("testing"),
				ProofOfPossession:           []byte("pop"),
				WithdrawalCredentialsHash32: []byte("withdraw"),
			},
			amount:    76706966,
			timestamp: 34394393,
		},
	}

	for _, tt := range tests {
		data, err := EncodeDepositData(tt.depositData, tt.amount, tt.timestamp)
		if err != nil {
			t.Fatalf("could not encode data %v", err)
		}

		decAmount, decTimestamp, err := DecodeDepositAmountAndTimeStamp(data)
		if err != nil {
			t.Fatalf("could not decode data %v", err)
		}

		if tt.amount != decAmount {
			t.Errorf("decoded amount not equal to given amount, %d : %d", decAmount, tt.amount)
		}

		if tt.timestamp != decTimestamp {
			t.Errorf("decoded timestamp not equal to given timestamp, %d : %d", decTimestamp, tt.timestamp)
		}
	}
}
