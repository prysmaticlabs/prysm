package consts

import "os"

const (
	// WalletDefaultDirName for accounts-v2.
	WalletDefaultDirName = ".prysm-wallet-v2"
	// PasswordsDefaultDirName where account passwords are stored.
	PasswordsDefaultDirName = ".prysm-wallet-v2-passwords"
	// KeymanagerConfigFileName for the keymanager used by the wallet: direct, derived, or remote.
	KeymanagerConfigFileName = "keymanageropts.json"
	// EncryptedSeedFileName for persisting a wallet's seed when using a derived keymanager.
	EncryptedSeedFileName = "seed.encrypted.json"
	// PasswordFileSuffix for passwords persisted as text to disk.
	PasswordFileSuffix = ".pass"
	// NumAccountWords for human-readable names in wallets using a direct keymanager.
	NumAccountWords = 3 // Number of words in account human-readable names.
	// AccountFilePermissions for accounts saved to disk.
	AccountFilePermissions = os.O_CREATE | os.O_RDWR
	// DirectoryPermissions for directories created under the wallet path.
	DirectoryPermissions = os.ModePerm
)
