# Sharding contracts

Generate contract bindings from the sharding package with go generate:

```bash

cd ..
go run ../../cmd/abigen/main.go --sol validator_manager.sol --pkg contracts --out validator_manager.go

```
