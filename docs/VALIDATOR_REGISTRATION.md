# Validator Registration Workflow

This doc summarizes the work flow of registering to become a validator in the beacon chain. The scope is within Ruby Release.

### Step 1: Deploy Validator Registration Contract on Mainchain

To deploy VRC, we can use [deployVRC](https://github.com/terenc3t/geth-sharding/tree/contract-util/contracts/deployVRC) utility.  
Once we get the VRC contract address, we can launch our beacon chain node
```
# Deploy VRC with keystore UTCJSON and password
go run deployVRC.go --UTCPath /path/to/your/keystore/UTCJSON --passwordFile /path/to/your/password.txt
# Deploy VRC with private key
go run deployVRC.go --privKey 8a6db3b30934439c9f71f1fa777019810fd538c9c1e396809bcf9fd5535e20ca

INFO[0039] New contract deployed at 0x559eDab2b5896C2Bc37951325666ed08CD41099d
```

### Step 2: Launch Beacon Chain Node
Launch beacon chain node with account holder's public key and the VRC address we just deployed
```
bazel run //beacon-chain --\
  --vrcaddr 0x527580dd995c0ab81d01f9993eb39166796877a1 \
  --pubkey aaace816cdab194b4bc6c0de3575ccf917a9b9ecfead263720968e0e1b45739c \
  --web3provider  ws://127.0.0.1:8546 \
  --datadir /path/to/your/datadir \
  --rpc-port 4000 \
```

### Step 3: Deposit 32 ETH

Send a transaction to the deposit function in VRC with 32 ETH and beacon chain node account holder's public key as argument.
Since the signature scheme and the logic to do the actual signing and validating isn't there yet, it's ok to use a random byte32 as pub key
as long as pub key matches the one you used to launch beacon chain node. (ex: aaace816cdab194b4bc6c0de3575ccf917a9b9ecfead263720968e0e1b45739c)


### Step 4: Wait for Deposit Tx Block to be Mined

After the deposit transaction gets mined, beacon chain node will report account holder has been registered. Congrats! Now, you are contributing to the security of Ethereum Serenity
```
INFO[0000] Starting beacon node
INFO[0000] Starting web3 PoW chain service at ws://127.0.0.1:8546
INFO[0152] Validator registered in VRC with public key: aaace816cdab194b4bc6c0de3575ccf917a9b9ecfead263720968e0e1b45739c
```
