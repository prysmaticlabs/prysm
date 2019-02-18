package helpers

import (
	"bytes"
	"encoding/binary"
	"fmt"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/ssz"
)

// EncodeDepositData converts a deposit input proto into an a byte slice
// of Simple Serialized deposit input followed by 8 bytes for a deposit value
// and 8 bytes for a unix timestamp, all in LittleEndian format.
func EncodeDepositData(
	depositInput *pb.DepositInput,
	depositValue uint64,
	depositTimestamp int64,
) ([]byte, error) {
	wBuf := new(bytes.Buffer)
	if err := ssz.Encode(wBuf, depositInput); err != nil {
		return nil, fmt.Errorf("failed to encode deposit input: %v", err)
	}
	encodedInput := wBuf.Bytes()
	depositData := make([]byte, 0, 512)
	value := make([]byte, 8)
	binary.LittleEndian.PutUint64(value, depositValue)
	timestamp := make([]byte, 8)
	binary.LittleEndian.PutUint64(timestamp, uint64(depositTimestamp))

	depositData = append(depositData, value...)
	depositData = append(depositData, timestamp...)
	depositData = append(depositData, encodedInput...)

	return depositData, nil
}

// DecodeDepositInput unmarshals a depositData byte slice into
// a proto *pb.DepositInput by using the Simple Serialize (SSZ)
// algorithm.
// TODO(#1253): Do not assume we will receive serialized proto objects - instead,
// replace completely by a common struct which can be simple serialized.
func DecodeDepositInput(depositData []byte) (*pb.DepositInput, error) {
	if len(depositData) < 16 {
		return nil, fmt.Errorf(
			"deposit data slice too small: len(depositData) = %d",
			len(depositData),
		)
	}
	depositInput := new(pb.DepositInput)
	// Since the value deposited and the timestamp are both 8 bytes each,
	// the deposit data is the chunk after the first 16 bytes.
	depositInputBytes := depositData[16:]
	rBuf := bytes.NewReader(depositInputBytes)
	if err := ssz.Decode(rBuf, depositInput); err != nil {
		return nil, fmt.Errorf("ssz decode failed: %v", err)
	}
	return depositInput, nil
}

// DecodeDepositAmountAndTimeStamp extracts the deposit amount and timestamp
// from the given deposit data.
func DecodeDepositAmountAndTimeStamp(depositData []byte) (uint64, int64, error) {
	// Last 16 bytes of deposit data are 8 bytes for value
	// and 8 bytes for timestamp. Everything before that is a
	// Simple Serialized deposit input value.
	if len(depositData) < 16 {
		return 0, 0, fmt.Errorf(
			"deposit data slice too small: len(depositData) = %d",
			len(depositData),
		)
	}

	// the amount occupies the first 8 bytes while the
	// timestamp occupies the next 8 bytes.
	amount := binary.LittleEndian.Uint64(depositData[:8])
	timestamp := binary.LittleEndian.Uint64(depositData[8:16])

	return amount, int64(timestamp), nil
}
