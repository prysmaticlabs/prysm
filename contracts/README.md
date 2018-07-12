## Validator Registration Contract

For beacon chain design, a validator will deposit 32 ETH to the main chain smart contract.
The deposit is considered to be burned. As you burn the 32 ETH to participate,
the beacon chain will see it and will credit the validator with the validator bond,
and the validator can begin to validate. At some point in the future, after a hard fork,
the original deposit + interest can be withdrawn back on one of the shards.

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