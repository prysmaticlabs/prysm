package payloadattestation

type MockVerifier struct {
	ErrCurrentSlot             error
	ErrAlreadySeenMessage      error
	ErrInvalidStatus           error
	ErrValidatorNotInPayload   error
	ErrBlockRootNotSeen        error
	ErrInvalidBlockRoot        error
	ErrInvalidMessageSignature error
	ErrUnsatisfiedRequirement  error
	cbVerifiedMessage          func() (VerifiedReadOnly, error)
}

func (m *MockVerifier) CheckCurrentSlot() error {
	return m.ErrCurrentSlot
}

func (m *MockVerifier) CheckPayloadMessageAlreadySeen() error {
	return m.ErrAlreadySeenMessage
}

func (m *MockVerifier) CheckKnownPayloadStatus() error {
	return m.ErrInvalidStatus
}

func (m *MockVerifier) CheckValidatorInPayload() error {
	return m.ErrValidatorNotInPayload
}

func (m *MockVerifier) CheckBlockRootSeen() error {
	return m.ErrBlockRootNotSeen
}

func (m *MockVerifier) CheckBlockRootValid() error {
	return m.ErrInvalidBlockRoot
}

func (m *MockVerifier) CheckSignatureValid() error {
	return m.ErrInvalidMessageSignature
}

func (m *MockVerifier) SatisfyRequirement(requirement Requirement) {}

func (m *MockVerifier) VerifiedPayloadAttestationMessage() (VerifiedReadOnly, error) {
	return m.cbVerifiedMessage()
}
