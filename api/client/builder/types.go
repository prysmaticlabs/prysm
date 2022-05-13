package builder

import (
	"encoding/json"
	"fmt"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

type SignedValidatorRegistration struct {
	*eth.SignedValidatorRegistrationV1
}

type ValidatorRegistration struct {
	*eth.ValidatorRegistrationV1
}

func (r *SignedValidatorRegistration) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct{
		Message *ValidatorRegistration `json:"message,omitempty"`
		Signature hexSlice `json:"signature,omitempty"`
	}{
		Message: &ValidatorRegistration{r.Message},
		Signature: r.SignedValidatorRegistrationV1.Signature,
	})
}

func (r *ValidatorRegistration) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct{
		FeeRecipient hexSlice `json:"fee_recipient,omitempty"`
		GasLimit string `json:"gas_limit"`
		Timestamp string `json:"timestamp"`
		Pubkey hexSlice `json:"pubkey,omitempty"`
		*eth.ValidatorRegistrationV1
	}{
		FeeRecipient: r.FeeRecipient,
		GasLimit: fmt.Sprintf("%d", r.GasLimit),
		Timestamp: fmt.Sprintf("%d", r.Timestamp),
		Pubkey: r.Pubkey,
		ValidatorRegistrationV1: r.ValidatorRegistrationV1,
	})
}

type hexSlice []byte

func (hs hexSlice) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprintf("%#x", hs)), nil
}

type BuilderBid struct {
	*eth.BuilderBid
}

type ExecHeaderResponse struct {
	*eth.SignedBuilderBid
}