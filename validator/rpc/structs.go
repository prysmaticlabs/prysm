package rpc

import (
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"
	"github.com/prysmaticlabs/prysm/v4/validator/keymanager"
)

// local keymanager api
type ListKeystoresResponse struct {
	Data []*Keystore `json:"data"`
}

type Keystore struct {
	ValidatingPubkey string `json:"validating_pubkey"`
	DerivationPath   string `json:"derivation_path"`
}

type ImportKeystoresRequest struct {
	Keystores          []string `json:"keystores"`
	Passwords          []string `json:"passwords"`
	SlashingProtection string   `json:"slashing_protection"`
}

type ImportKeystoresResponse struct {
	Data []*keymanager.KeyStatus `json:"data"`
}

type DeleteKeystoresRequest struct {
	Pubkeys []string `json:"pubkeys"`
}

type DeleteKeystoresResponse struct {
	Data               []*keymanager.KeyStatus `json:"data"`
	SlashingProtection string                  `json:"slashing_protection"`
}

// voluntary exit keymanager api
type SetVoluntaryExitResponse struct {
	Data *shared.SignedVoluntaryExit `json:"data"`
}

// gas limit keymanager api
type GasLimitMetaData struct {
	Pubkey   string `json:"pubkey"`
	GasLimit string `json:"gas_limit"`
}

type GetGasLimitResponse struct {
	Data *GasLimitMetaData `json:"data"`
}

type SetGasLimitRequest struct {
	GasLimit string `json:"gas_limit"`
}

// remote keymanager api
type ListRemoteKeysResponse struct {
	Data []*RemoteKey `json:"data"`
}

type RemoteKey struct {
	Pubkey   string `json:"pubkey"`
	Url      string `json:"url"`
	Readonly bool   `json:"readonly"`
}

type ImportRemoteKeysRequest struct {
	RemoteKeys []*RemoteKey `json:"remote_keys"`
}

type DeleteRemoteKeysRequest struct {
	Pubkeys []string `json:"pubkeys"`
}

type RemoteKeysResponse struct {
	Data []*keymanager.KeyStatus `json:"data"`
}

// Fee Recipient keymanager api
type FeeRecipient struct {
	Pubkey     string `json:"pubkey"`
	Ethaddress string `json:"ethaddress"`
}

type GetFeeRecipientByPubkeyResponse struct {
	Data *FeeRecipient `json:"data"`
}

type SetFeeRecipientByPubkeyRequest struct {
	Ethaddress string `json:"ethaddress"`
}

type BeaconStatusResponse struct {
	BeaconNodeEndpoint     string            `json:"beacon_node_endpoint"`
	Connected              bool              `json:"connected"`
	Syncing                bool              `json:"syncing"`
	GenesisTime            string            `json:"genesis_time"`
	DepositContractAddress string            `json:"deposit_contract_address"`
	ChainHead              *shared.ChainHead `json:"chain_head"`
}

// KeymanagerKind is a type of key manager for the wallet
type KeymanagerKind string

const (
	derivedKeymanagerKind    KeymanagerKind = "DERIVED"
	importedKeymanagerKind   KeymanagerKind = "IMPORTED"
	web3signerKeymanagerKind KeymanagerKind = "WEB3SIGNER"
)

type CreateWalletRequest struct {
	Keymanager       KeymanagerKind `json:"keymanager"`
	WalletPassword   string         `json:"wallet_password"`
	Mnemonic         string         `json:"mnemonic"`
	NumAccounts      uint64         `json:"num_accounts"`
	MnemonicLanguage string         `json:"mnemonic_language"`
}

type CreateWalletResponse struct {
	Wallet *WalletResponse `json:"wallet"`
}

type GenerateMnemonicResponse struct {
	Mnemonic string `json:"mnemonic"`
}

type WalletResponse struct {
	WalletPath     string         `json:"wallet_path"`
	KeymanagerKind KeymanagerKind `json:"keymanager_kind"`
}

type ValidateKeystoresRequest struct {
	Keystores         []string `json:"keystores"`
	KeystoresPassword string   `json:"keystores_password"`
}

type RecoverWalletRequest struct {
	Mnemonic         string `json:"mnemonic"`
	NumAccounts      uint64 `json:"num_accounts"`
	WalletPassword   string `json:"wallet_password"`
	Language         string `json:"language"`
	Mnemonic25ThWord string `json:"mnemonic25th_word"`
}

type ImportSlashingProtectionRequest struct {
	SlashingProtectionJson string `json:"slashing_protection_json"`
}

type ExportSlashingProtectionResponse struct {
	File string `json:"file"`
}

type BackupAccountsRequest struct {
	PublicKeys     []string `json:"public_keys"`
	BackupPassword string   `json:"backup_password"`
}

type VoluntaryExitRequest struct {
	PublicKeys []string `json:"public_keys"`
}

type BackupAccountsResponse struct {
	ZipFile string `json:"zip_file"`
}

type ListAccountsResponse struct {
	Accounts      []*Account `json:"accounts"`
	NextPageToken string     `json:"next_page_token"`
	TotalSize     int32      `json:"total_size"`
}

type Account struct {
	ValidatingPublicKey string `json:"validating_public_key"`
	AccountName         string `json:"account_name"`
	DepositTxData       string `json:"deposit_tx_data"`
	DerivationPath      string `json:"derivation_path"`
}

type VoluntaryExitResponse struct {
	ExitedKeys [][]byte `protobuf:"bytes,1,rep,name=exited_keys,json=exitedKeys,proto3" json:"exited_keys,omitempty"`
}

type InitializeAuthResponse struct {
	HasSignedUp bool `json:"has_signed_up"`
	HasWallet   bool `json:"has_wallet"`
}
