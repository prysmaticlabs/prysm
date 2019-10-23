# Testnet 

The Prysmatic Labs test network is available for anyone to join. The easiest way to participate is by joining through the website, https://prylabs.net.

## Interop

For developers looking to connect a client other than Prysm to the test network, here is the relevant information for compatability. 


**Spec version** - [v0.8.3](https://github.com/ethereum/eth2.0-specs/tree/v0.8.3)

**ETH 1 Deposit Contract Address** - See https://prylabs.net/contract. This contract is deployed on the [goerli](https://goerli.net/) network. 

**Genesis time** - The ETH1 block time in which the 64th deposit to start ETH2 was included. This is NOT midnight of the next day as required by spec.

### ETH 2 Configuration

Use the [minimal config](https://github.com/ethereum/eth2.0-specs/blob/v0.8.3/configs/minimal.yaml) with the following changes.

| field | value |
|-------|-------|
| MIN_DEPOSIT_AMOUNT | 100 |
| MAX_EFFECTIVE_BALANCE | 3.2 * 1e9 |
| EJECTION_BALANCE | 1.6 * 1e9 |
| EFFECTIVE_BALANCE_INCREMENT | 0.1 * 1e9 |
| ETH1_FOLLOW_DISTANCE | 16 | 
| GENESIS_FORK_VERSION | See [latest code](https://github.com/prysmaticlabs/prysm/blob/master/shared/params/config.go#L236) |

These parameters reduce the minimal config to 1/10 of the required ETH.

We have a genesis.ssz file available for download [here](https://prysmaticlabs.com/uploads/genesis.ssz)

### Connecting to the network

We have a libp2p bootstrap node available at `/dns4/prylabs.net/tcp/30001/p2p/16Uiu2HAm7Qwe19vz9WzD2Mxn7fXd1vgHHp4iccuyq7TxwRXoAGfc`.

Some of the Prysmatic Labs hosted nodes are behind a libp2p relay, so your libp2p implementation protocol should understand this functionality.

### Other

Undoubtably, you will have bugs. Reach out to us on [Discord](https://discord.gg/KSA7rPr) and be sure to capture issues on Github at https://github.com/prysmaticlabs/prysm/issues. 

If you have instructions for you client, we would love to attempt this on your behalf. Kindly send over the instructions via github issue, PR, email to team@prysmaticlabs.com, or discord.
