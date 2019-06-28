package helpers

import (
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/go-ssz"
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

func TestSSZUnmarshalDepositInput(t *testing.T) {

	depositData :=[219]byte{0, 32, 188, 190, 0, 0, 0, 0, 124, 26, 9 ,93, 0, 0, 0, 0, 188, 0, 0, 0, 48, 0, 0, 0, 135, 33, 178, 215, 169, 185, 93, 20, 57, 117, 186, 21, 8, 219, 127, 49, 0, 150, 117, 121, 17, 213, 17, 63, 184, 76, 250, 231, 158, 185, 82, 221, 122, 169, 68, 116, 252, 48, 117, 244, 167, 243, 39, 135, 210, 213, 144, 252, 96, 0, 0, 0, 137, 33, 84, 14, 215, 15, 192, 157, 117, 121, 27, 154, 231, 61, 232, 61, 102, 153, 164, 125, 217, 131, 85, 182, 235, 80, 235, 159, 168, 178, 11, 196, 84, 3, 216, 235, 109, 122, 175, 63, 134, 114, 88, 146, 46 ,232, 144, 196, 10, 64, 148, 198, 57, 154, 225, 81, 222, 190, 54, 56, 87, 29, 206, 232, 112, 38, 32, 119, 75, 254, 153, 142, 168, 70, 152, 218, 2, 240, 93, 228, 221, 133, 1, 200, 204, 96, 246, 0 ,33, 160, 43 ,207, 45, 15, 35, 168, 32, 0, 0, 0, 0, 57, 187, 69, 48, 82, 158, 110, 143, 8, 66, 63, 143, 244, 215, 104, 9, 225, 31, 187, 40, 251, 230, 38, 236, 52, 117, 230, 50, 204, 41, 187}
	depositInput := new(pb.DepositInput)
	
	// Since the value deposited and the timestamp are both 8 bytes each,
	// the deposit data is the chunk after the first 16 bytes.
	depositInputBytes := depositData[16:]
	err := ssz.Unmarshal(depositInputBytes, &depositInput)
	if err != nil {
		t.Fatalf("TestSSZUnmarshal failed : %v", err)
	}
}
