# Kubernetes Configuration for Ethereum Serenity

## Requirements

- Kubernetes v1.11+ (for PriorityClass)
- Minikube (for now)

### Starting minikube with v1.11

As of minikube 0.28.2, the default version of kubernetes is 1.10.0. In order to
start a local cluster with v1.11.0, run the following:

```
minikube start --kubernetes-version=v1.11.0 --cpus 4
```


### Istio

Be sure to update the loadBalancerIP and includeIPRanges as needed. Refer to
[this guide](https://github.com/stefanprodan/istio-gke/blob/master/docs/istio/04-istio-setup.md) 
if necessary.

First download the release
```
curl -L https://git.io/getLatestIstio | sh -
```

```
helm install install/kubernetes/helm/istio --name istio --namespace istio-system --set grafana.enabled=true --set tracing.enabled=true --set gateways.istio-ingressgateway.loadBalancerIP=35.224.249.2 --set global.proxy.includeIPRanges="10.52.0.0/14\,10.55.240.0/20" --set kiali.enabled=true --set telemetry-gateway.grafanaEnabled=true 
```

### Geth's Genesis file

This file is the default provided by geth-genesis secret. 

```json
{                                                                               
    "config": {                                                                 
        "chainId": 1337,                                                        
        "homesteadBlock": 0,                                                    
        "eip155Block": 0,                                                       
        "eip158Block": 0                                                        
    },                                                                          
    "difficulty": "0x0",                                                      
    "gasLimit": "0x2100000",                                                    
    "alloc": {                                                                  
       "717c3a6e4cbd476c2312612155eb233bf498dd5b":                              
         { "balance": "0x1337000000000000000000" }                             
    }                                                                           
}
```

The private key for the allocation above is:

```text
783da8ef5343c3019748506305d400bca8c324a5819f3a7f7fbf0c0a0d799b09
```

NOTE: Obviously, do not use this wallet key for anything with real money on it!

To update the genesis secret, change value in geth/genesis.secret.yaml to the
base64 encoded string for the genesis.json.

Example:

```bash
cat /tmp/genesis.json | json-minify | base64
```

### Deploying Geth Mainchain

First, launch the bootnode so that geth nodes can discover each other.

```bash
bazel run //k8s/geth:bootnode.deploy.apply
```

Then launch everything else.

```bash
bazel run //k8s:everything.apply
```

This creates a few nodes and one miner with CPU restrictions. After ~30 
minutes, the miner has generated the DAG and begins mining. The miners have a
stateful volume for their DAGs so that they do not have to regenerate them on
restart. 

Note: DAG generation time can be improved by giving the miner more CPU in the 
deployment yaml.

### Bootstrapping the Beacon Chain

TODO: This process is currently manual and needs to be improved!

Using the private key above and the deployContract tool, deploy the validator
registration contract.

```bash
# get the address the node service
minikube service geth-nodes --url
```

Example response:

```bash
http://192.168.99.100:30051
http://192.168.99.100:31745
```

Using the first port provided (RPC). Run the deployContract tool

```
bazel run //contracts/deposit-contract/deployContract --\
  --privKey=783da8ef5343c3019748506305d400bca8c324a5819f3a7f7fbf0c0a0d799b09 \
  --httpPath=http://192.168.99.100:30051
```

Example output:

```
INFO main: New contract deployed address=0x541AfaC5266c534de039B4A1a53519e76ea82846
```

Set this value for the deposit contract addr flag in 
k8s/beacon-chain/beacon-chain.deploy.yaml.

Ensure that the beacon-chain and client docker images are up to date. 

```bash
bazel run //beacon-chain:push_image
bazel run //client:push_image
```

Start the beacon chain nodes

```bash
bazel run //k8s/beacon-chain:everything.apply
```

Start the clients

```bash
bazel run //k8s/client:everything.apply
```

### Accessing Geth Services

Check out the ethstats dashboard by querying minikube for the service URL.

```bash
minikube service geth-ethstats --url
```

Accessing the geth nodes.

```bash
minikube service geth-nodes --url

# Example output
http://192.168.99.100:30451
http://192.168.99.100:32164
```

The first URL will be the rpc endpoint and the second URL will be the websocket
endpoint.

So we can use these values locally to connect to our local cluster.

```bash
bazel run //beacon-chain -- --web3provider=ws://192.168.99.100:32164
```

