## Prysm Internal Validator Deposit Contract

**NOTE: THIS IS NOT THE OFFICIAL ETHEREUM VALIDATOR DEPOSIT CONTRACT. THE OFFICIAL CONTRACT CAN ONLY BE FOUND [HERE](https://github.com/ethereum/eth2.0-specs/blob/e4a9c5fa29def20c4264cd860868f131d6f40e72/solidity_deposit_contract/deposit_contract.sol). THE ONLY DEPOSIT CONTRACT ON MAINNET HAS ADDRESS 0x00000000219ab540356cbb839cbe05303d7705fa. DO NOT USE THE CONTRACT IN THIS FOLDER OUTSIDE OF DEVELOPMENT**

## How to execute tests

```
bazel test //contracts/deposit-contract:go_default_test

```

Run with `-v` option for detailed log output

```
bazel test //contracts/deposit-contract:go_default_test --test_arg=-test.v --test_output=streamed 
=== RUN   TestSetupRegistrationContract_OK
--- PASS: TestSetupRegistrationContract_OK (0.07s)
=== RUN   TestRegister_Below1ETH
--- PASS: TestRegister_Below1ETH (0.02s)
=== RUN   TestRegister_Above32Eth
--- PASS: TestRegister_Above32Eth (0.02s)
=== RUN   TestValidatorRegister_OK
--- PASS: TestValidatorRegister_OK (0.08s)
=== RUN   TestDrain
--- PASS: TestDrain (0.04s)
PASS
ok      contracts/deposit-contract        0.633s
```
