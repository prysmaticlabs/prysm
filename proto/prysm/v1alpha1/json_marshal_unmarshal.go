package eth

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

type blobJSON struct {
	Blob []hexutil.Bytes `json:"blob"`
}

// MarshalJSON --
func (b *Blob) MarshalJSON() ([]byte, error) {
	hexBlob := make([]hexutil.Bytes, len(b.Blob))
	for i, b := range b.Blob {
		hexBlob[i] = b
	}
	return json.Marshal(blobJSON{
		Blob: hexBlob,
	})
}

// UnmarshalJSON --
func (b *Blob) UnmarshalJSON(enc []byte) error {
	dec := blobJSON{}
	if err := json.Unmarshal(enc, &dec); err != nil {
		return err
	}
	*b = Blob{}
	decodedBlobBytes := make([][]byte, len(dec.Blob))
	for i, b := range dec.Blob {
		decodedBlobBytes[i] = b
	}
	b.Blob = decodedBlobBytes
	return nil
}
