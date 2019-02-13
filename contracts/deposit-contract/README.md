## Deposit Contract

A validator will deposit 32 ETH to the deposit
contract. The contract will generate a log showing the validator as a
qualified validator. 
The deposit is considered to be burned. As you burn the 32 ETH to participate,
the beacon chain will see it and will credit the validator with the validator bond,
At some point in the future, after a hard fork,
the original deposit + interest can be withdrawn back on one of the shards.  
To call the `registration` function, it takes arguments of `pubkey`, 
`proof_of_possession`, `withdrawal_credentials`.
If the user wants to deposit more than `DEPOSIT_SIZE` ETH, they would
need to make multiple `deposit` calls.  
When the contract publishes the `ChainStart` log, beacon nodes will
start off the beacon chain with slot 0 and last recorded `block.timestamp`
as beacon chain genesis time.
The registration contract generate logs with the various arguments
for consumption by beacon nodes. It does not validate `proof_of_possession`
and `withdrawal_credentials`, pushing the validation logic to the
beacon chain.

## How to generate bindings for vyper contract

This requires that you have vyper and abigen installed in your local machine.
Vyper: https://github.com/ethereum/vyper
Abigen: https://github.com/ethereum/go-ethereum/tree/master/cmd/abigen

To generate the abi using the vyper compiler, you can use

```

vyper -f abi  ./depositContract.v.py

```

Then the abi will be outputted and you can save it in `abi.json` in the folder. 

To generate the bytecode you can then use 

```

vyper  ./depositContract.v.py

```

and save the bytecode in `bytecode.bin` in the folder. Now with both the abi and bytecode
we can generate the go bindings. 

```
abigen -bin ./bytecode.bin -abi ./abi.json -out ./depositContract.go --pkg depositContract

```

## How to execute tests

```
go test ./...

```

Run with `-v` option for detailed log output

```
go test ./... -v
=== RUN   TestSetupAndContractRegistration
--- PASS: TestSetupAndContractRegistration (0.01s)
=== RUN   TestRegisterWithLessThan32Eth
--- PASS: TestRegisterWithLessThan32Eth (0.00s)
=== RUN   TestRegisterWithMoreThan32Eth
--- PASS: TestRegisterWithMoreThan32Eth (0.00s)
=== RUN   TestRegisterTwice
--- PASS: TestRegisterTwice (0.01s)
=== RUN   TestRegister
--- PASS: TestRegister (0.01s)
PASS
ok      beacon-chain/contracts/deposit-contract/  0.151s
```