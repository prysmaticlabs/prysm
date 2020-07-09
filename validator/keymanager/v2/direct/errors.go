package direct

const (
	// ErrCouldNotReadPassword returns when the inputted password could not be read.
	ErrCouldNotReadPassword = "could not read password"
	// ErrCouldNotReadKeystore returns when the given keystore file cannot not be read.
	ErrCouldNotReadKeystore = "could not read keystore file"
	// ErrCouldNotDecodeJSON returns when a keystore cannot be decoded from given bytes.
	ErrCouldNotDecodeJSON = "could not decode keystore json"
	// ErrCouldNotDecryptSigningKey returns when a key cannot be decrypted with the given password.
	ErrCouldNotDecryptSigningKey = "could not decrypt validator signing key"
	// ErrCouldNotInstantiateBLSSecretKey returns when a given BLS key can't be retrieved from bytes.
	ErrCouldNotInstantiateBLSSecretKey = "could not instantiate bls secret key from bytes"
)
