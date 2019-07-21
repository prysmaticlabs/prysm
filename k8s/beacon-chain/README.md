# Minikube setup for Beacon-chain and 8 Validators

## Set up Kubernetes and Minikube

### Make sure you have kubectl installed

https://kubernetes.io/docs/tasks/tools/install-minikube/

### For installing the hypervisor, it's recommended if you're on OS X to use hyperkit, install it here

https://github.com/kubernetes/minikube/blob/master/docs/drivers.md#hyperkit-driver

### Configure Hyperkit as your default

`minikube config set vm-driver hyperkit`

## Running the beacon chain and validators in Minikube

### Start Minikube and the dashboard

`minikube start --memory 4096`

`minikube dashboard`

### Set up the beacon-chain config

Deploy a new deposit contract with:

```
bazel run //contracts/deposit-contract/deployContract -- --httpPath=https://goerli.prylabs.net --privKey=$(cat /path/to/private/key/file) --chainStart=8 --minDeposit=100000 --maxDeposit=3200000 --customChainstartDelay 120
```

Place the Goerli network deposit contract address from above in `k8s/beacon-chain/beacon-config.config.yaml`

### Apply the namespace and config yamls

```
cd k8s/
kubectl apply -f priority.yaml

cd beacon-chain/
kubectl apply -f namespace.yaml
kubectl apply -f beacon-config.config.yaml
```

### Edit the beacon-chain.deploy.yaml to prepare it for a local instance

Change the lines 40-53 to:

```
  args:
    - --web3provider=ws://goerli.prylabs.com/websocket
    #- --verbosity=debug
    - --deposit-contract=$(DEPOSIT_CONTRACT_ADDRESS)
    - --rpc-port=4000
    - --monitoring-port=9090
#    - --bootstrap-node=/ip4/$(BOOTNODE_SERVICE_HOST)/tcp/$(BOOTNODE_SERVICE_PORT)/p2p/QmQEe7o6hKJdGdSkJRh7WJzS6xrex5f4w2SPR6oWbJNriw
#    - --relay-node=/ip4/35.224.249.2/tcp/30000/p2p/QmfAgkmjiZNZhr2wFN9TwaRgHouMTBT6HELyzE5A3BT2wK
    - --p2p-port=5000
    - --demo-config
#    - --enable-tracing
#    - --tracing-endpoint=http://jaeger-collector.istio-system.svc.cluster.local:14268
#    - --trace-sample-fraction=1.0
    - --datadir=/data
```

(commenting out the 3 tracing items and the bootstrap/relay nodes, also changing the web3 provider to ws://goerli.prylabs.com/websocket)

### Apply the beacon chain yamls to start the beacon-chain

```
kubectl apply -f beacon-chain.service.yaml
kubectl apply -f beacon-chain.deploy.yaml
```

### Go into the minikube dashboard and set your namespace to "beacon-chain" in the middle of the left side

You should see 3 beacon-chain node replicas

### Add your private key into the cluster manger config

Convert a goerli private key with ETH to base64 in [a browser js console](https://stackoverflow.com/questions/246801/how-can-you-encode-a-string-to-base64-in-javascript) and set it in `k8s/beacon-chain/cluster-manager.encrypted_secret.yaml`.

### Edit the cluster-manager.yaml to use the Prylabs Goerli node endpoint

Change lines 55-63 to:

```
  args:
  - --deposit-contract=$(DEPOSIT_CONTRACT_ADDRESS)
  - --private-key=$(PRIVATE_KEY)
  - --rpc=ws://goerli.prylabs.com/websocket
  - --port=8000
  - --metrics-port=9090
  - --deposit-amount=3200000000000000
  - --db-path=/data
  - --verbose
```

### Edit the validator.deploy.yaml to prepare it for a local instance

Change the lines 20-28 to:

```
  args:
  - --keystore-path=/keystore
  - --password=nopass
  - --datadir=/data
  - --beacon-rpc-provider=beacon-chain:4000
  - --demo-config
#  - --enable-tracing
#  - --tracing-endpoint=http://jaeger-collector.istio-system.svc.cluster.local:14268
#  - --trace-sample-fraction=1.0
```

(commenting out the bottom 3)

### Apply the cluster manager and validator yamls

```
kubectl apply -f cluster-manager.encrypted_secret.yaml
kubectl apply -f cluster-manager.yaml
kubectl apply -f validator.deploy.yaml
```

### Check the beacon-chain namespace in the dashboard and you should see 8 validators loading

# Minikube common commands

## To remove all the containers inside the minikube

`kubectl delete -f k8s/beacon-chain/namespace.yaml`

## To stop the minikube instance

`minikube stop`

## To delete the container

`minikube delete`
