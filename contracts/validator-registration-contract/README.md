## Validator Registration Contract

A validator will deposit 32 ETH to the registration
contract. The contract will generate a receipt showing the validator as a
qualified validator. 
The deposit is considered to be burned. As you burn the 32 ETH to participate,
the beacon chain will see it and will credit the validator with the validator bond,
At some point in the future, after a hard fork,
the original deposit + interest can be withdrawn back on one of the shards.  
To call the `registration` function, it takes arguments of `pubkey`, 
`proof_of_possession`, `withdrawal_credentials` and `randao_commitment`. 
If the user wants to deposit more than `DEPOSIT_SIZE` ETH, they would
need to make multiple `registration` calls.  
When the contract publishes the `ChainStart` log, beacon nodes will
start off the beacon chain with slot 0 and last recorded `block.timestamp`
as beacon chain genesis time.
The registration contract generate receipts with the various arguments
for consumption by beacon nodes. It does not validate `proof_of_possession`
and `withdrawal_credentials`, pushing the validation logic to the
beacon chain.

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
ok      beacon-chain/contracts  0.151s
```