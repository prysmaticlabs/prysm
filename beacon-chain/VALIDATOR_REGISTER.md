# Validator Registration Workflow

This doc will summarize the work flow of registering to become a validator in the beacon chain. The scope is within Ruby Release.

### Step 1: Deploy validator registration contract if it hasn't been done
To deploy VRC, we can use [deployVRC](https://github.com/terenc3t/geth-sharding/tree/contract-util/contracts/deployVRC) utility
Once we get the VRC contract address, we can launch our beacon chain node
```
go run deployVRC.go --privKey 8a6db3b30934439c9f71f1fa777019810fd538c9c1e396809bcf9fd5535e20ca
INFO[0039] New contract deployed at 0x559eDab2b5896C2Bc37951325666ed08CD41099d
```

### Step 2: Launch beacon chain node
Launch beacon chain node with account holder's public key and just deployed VRC address
```
./bazel-bin/path/to/your/beacon-chain/binary --vrcaddr 0x527580dd995c0ab81d01f9993eb39166796877a1 --pubkey aaace816cdab194b4bc6c0de3575ccf917a9b9ecfead263720968e0e1b45739c

```

### Step 3: Send a transaction to VRC with 32 ETH and public key corresponded to beacon chain node account holder's public key
Use Remix for example, see the following screen shot


### Step 4: After the transaction gets mined, beacon chain node will report account holder has been registered. Congrats!
```
INFO[0000] Starting beacon node
INFO[0000] Starting web3 PoW chain service at ws://127.0.0.1:8546
INFO[0152] Validator registered in VRC with public key: aaace816cdab194b4bc6c0de3575ccf917a9b9ecfead263720968e0e1b45739c
```

