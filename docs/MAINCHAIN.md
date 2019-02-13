# Connecting Ethereum Serenity to a Mainchain, Go-Ethereum Node

In order to test out validator registration and synchronize the Ethereum Serenity beacon chain with a proof-of-work, mainchain node, you'll need to initialize your own private Ethereum blockchain as follows:

## Running a Local Geth Node

To start a local Geth node, you can create your own `genesis.json` file similar to:

```json
{
    "config": {
        "chainId": 12345,
        "homesteadBlock": 0,
        "eip155Block": 0,
        "eip158Block": 0
    },
    "difficulty": "200",
    "gasLimit": "210000000000",
    "alloc": {
        "826f3F66dB0416ea82033aE917A611bfBF4D98b6": { "balance": "300000" }
    }
}
```

The `alloc` portion specifies account addresses with prefunded ETH when the Ethereum blockchain is created. You can modify this section of the genesis to include your own test address and prefund it with 100ETH.

Then, you can build and init a new instance of a local, Ethereum blockchain as follows:

```
geth init /path/to/genesis.json --datadir /path/to/your/datadir
geth --nodiscover console --datadir /path/to/your/datadir --networkid 12345 --ws --wsaddr=127.0.0.1 --wsport 8546 --wsorigins "*" --rpc
````

It is **important** to note that the `--networkid` flag must match the `chainId` property in the genesis file.

Then, the geth console can start up and you can start a miner as follows:

    > personal.newAccount()
    > miner.setEtherbase(eth.accounts[0])
    > miner.start(1)

Now, save the passphrase you used in the geth node into a text file called password.txt.

## Build Our Beacon-Chain + Validator System

Build the beacon chain and validator projects as follows:

```
bazel build //beacon-chain:beacon-chain
bazel build //validator:validator
```

## Deploy a Validator Registation Contract

Deploy the Validator Registration Contract into the chain of the running geth node by following the instructions [here](https://github.com/prysmaticlabs/prysm/blob/master/contracts/deposit-contract/deployContract/README.md).

## Running a Beacon Node as a Validator

Make sure a geth node is running as a separate process according to the instructions from the previous section. Then, you can run a full beacon node as follows:

```
bazel run //beacon-chain --\
  --enable-powchain \
  --datadir /path/to/your/datadir \
  --genesis-json /path/to/your/genesis.json \
  --rpc-port 4000 \
  --verbosty debug
```

This will spin up a full beacon node that connects to your running geth node, opens up an RPC connection for sharding validators to connect to it, and begins listening for p2p events. Run the system at debug level log verbosity with `--verbosity debug` to see everything happening underneath the hood.

Now, deposit ETH to become a validator in the contract using instructions [here](https://github.com/prysmaticlabs/prysm/blob/master/docs/VALIDATOR_REGISTRATION.md).

