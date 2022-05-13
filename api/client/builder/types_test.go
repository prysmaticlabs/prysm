package builder

import (
	"encoding/json"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
	"testing"
)

func TestSignedValidatorRegistration_MarshalJSON(t *testing.T) {
	svr := &eth.SignedValidatorRegistrationV1{
		Message:   &eth.ValidatorRegistrationV1{
			FeeRecipient: make([]byte, 20),
			GasLimit:     0,
			Timestamp:    0,
			Pubkey:       make([]byte, 48),
		},
		Signature: make([]byte, 96),
	}
	je, err := json.Marshal(&SignedValidatorRegistration{SignedValidatorRegistrationV1: svr})
	require.NoError(t, err)
	// decode with a struct w/ plain strings so we can check the string encoding of the hex fields
	un := struct{
		Message struct {
			FeeRecipient string `json:"fee_recipient"`
			Pubkey string `json:"pubkey"`
		} `json:"message"`
		Signature string `json:"signature"`
	}{}
	require.NoError(t, json.Unmarshal(je, &un))
	require.Equal(t, "0x000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000", un.Signature)
	require.Equal(t, "0x0000000000000000000000000000000000000000", un.Message.FeeRecipient)
	require.Equal(t, "0x000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000", un.Message.Pubkey)
}

var testExampleHeaderResponse = `{
  "version": "bellatrix",
  "data": {
    "message": {
      "header": {
        "parent_hash": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
        "fee_recipient": "0xabcf8e0d4e9587369b2301d0790347320302cc09",
        "state_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
        "receipts_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
        "logs_bloom": "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
        "prev_randao": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
        "block_number": "1",
        "gas_limit": "1",
        "gas_used": "1",
        "timestamp": "1",
        "extra_data": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
        "base_fee_per_gas": "1",
        "block_hash": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
        "transactions_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
      },
      "value": "1",
      "pubkey": "0x93247f2209abcacf57b75a51dafae777f9dd38bc7053d1af526f220a7489a6d3a2753e5f3e8b1cfe39b56f43611df74a"
    },
    "signature": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"
  }
}`

func TestBuilderBidUnmarshal(t *testing.T) {
	bb := &BuilderBid{}
	require.NoError(t, json.Unmarshal([]byte(testExampleHeaderResponse), bb))
}
