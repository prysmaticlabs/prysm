// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

// Package utils contains internal helper functions for go-ethereum commands.
package utils

import (
	"math/big"

	"github.com/ethereum/go-ethereum/node"
	shardparams "github.com/prysmaticlabs/geth-sharding/sharding/params"
	"github.com/urfave/cli"
)

var (
	// General settings
	IPCPathFlag = DirectoryFlag{
		Name:  "ipcpath",
		Usage: "Filename for IPC socket/pipe within the datadir (explicit paths escape it)",
	}
	DataDirFlag = DirectoryFlag{
		Name:  "datadir",
		Usage: "Data directory for the databases and keystore",
		Value: DirectoryString{node.DefaultDataDir()},
	}
	NetworkIdFlag = cli.Uint64Flag{
		Name:  "networkid",
		Usage: "Network identifier (integer, 1=Frontier, 2=Morden (disused), 3=Ropsten, 4=Rinkeby)",
		Value: 1,
	}
	PasswordFileFlag = cli.StringFlag{
		Name:  "password",
		Usage: "Password file to use for non-interactive password input",
		Value: "",
	}
	// Sharding Settings
	DepositFlag = cli.BoolFlag{
		Name:  "deposit",
		Usage: "To become a notary in a sharding node, " + new(big.Int).Div(shardparams.DefaultConfig.NotaryDeposit, new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)).String() + " ETH will be deposited into SMC",
	}
	ActorFlag = cli.StringFlag{
		Name:  "actor",
		Usage: `use the --actor notary or --actor proposer to start a notary or proposer service in the sharding node. If omitted, the sharding node registers an Observer service that simply observes the activity in the sharded network`,
	}
	ShardIDFlag = cli.IntFlag{
		Name:  "shardid",
		Usage: `use the --shardid to determine which shard to start p2p server, listen for incoming transactions and perform proposer/observer duties`,
	}
)
