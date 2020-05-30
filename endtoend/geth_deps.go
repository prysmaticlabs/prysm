package endtoend

// This file contains the dependencies required for github.com/ethereum/go-ethereum/cmd/geth.
// Having these dependencies listed here helps go mod understand that these dependencies are
// necessary for end to end tests since we build go-ethereum binary for this test.
import (
	_ "github.com/ethereum/go-ethereum/accounts"
	_ "github.com/ethereum/go-ethereum/accounts/keystore"
	_ "github.com/ethereum/go-ethereum/cmd/utils"
	_ "github.com/ethereum/go-ethereum/common"
	_ "github.com/ethereum/go-ethereum/console"
	_ "github.com/ethereum/go-ethereum/eth"
	_ "github.com/ethereum/go-ethereum/eth/downloader"
	_ "github.com/ethereum/go-ethereum/ethclient"
	_ "github.com/ethereum/go-ethereum/les"
	_ "github.com/ethereum/go-ethereum/log"
	_ "github.com/ethereum/go-ethereum/metrics"
	_ "github.com/ethereum/go-ethereum/node"
)
