- install wsl 2 on your windows machine, verify it's wsl 2 and not 1

> sudo apt install npm

- can't install npm

> export PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin 

- then follow guidance here https://docs.prylabs.network/docs/install/install-with-bazel/
- version mismatch error, even when forcing the .bazelversion version to be installed
- delete .bazelversion, just make sure you keep WSL's bazel version in sync with the one specified in .bazelversion

> sudo apt install bazel --version=5.0.0
> bazel build //beacon-chain:beacon-chain --config=release

- workspace_status.sh not found
- use --workspace_status_command=/bin/true per https://docs.bazel.build/versions/main/user-manual.html#workspace_status 

> bazel build //cmd/beacon-chain:beacon-chain --config=release --workspace_status_command=/bin/true

- error related to https://stackoverflow.com/questions/48674104/clang-error-while-loading-shared-libraries-libtinfo-so-5-cannot-open-shared-o

> sudo apt install libncurses5
> bazel build //cmd/beacon-chain:beacon-chain --config=release --workspace_status_command=/bin/true

- 'gmpxx.h' file not found

> sudo apt-get install libgmp-dev
> bazel build //cmd/beacon-chain:beacon-chain --config=release --workspace_status_command=/bin/true

works!

start geth on localhost:8545

> bazel run //beacon-chain --config=release -- --http-web3provider=http://localhost:8545

works!

open vs code with prysm code

if you see errors, make sure you install https://www.mingw-w64.org/downloads/#mingw-builds

I used https://www.msys2.org/ and followed the instructions on that page
in combination with instructions on this page https://code.visualstudio.com/docs/cpp/config-mingw#_prerequisites 

blst errors... try --define=blst_modern=true

bazel build //validator:validator --config=release --workspace_status_command=/bin/true --define=blst_modern=true

-----

> bazel build //cmd/beacon-chain:beacon-chain  --workspace_status_command=/bin/true --define=blst_modern=true
> bazel build //cmd/validator:validator  --workspace_status_command=/bin/true --define=blst_modern=true

---

then issue this command bazel run //beacon-chain -- --datadir /tmp/chaindata --force-clear-db --interop-genesis-state /tmp/genesis.ssz --interop-eth1data-votes --min-sync-peers=0 --http-web3provider=http://localhost:8545 --deposit-contract 0x8A04d14125D0FDCDc742F4A05C051De07232EDa4  --bootstrap-node= --chain-config-file=/tmp/merge.yml 


