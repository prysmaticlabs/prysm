# Kubernetes 

## Requirements

- Kubernetes v1.11+ (for PriorityClass)
- Minikube (for now)

### Starting minikube with v1.11

As of minikube 0.28.2, the default version of kubernetes is 1.10.0. In order to
start a local cluster with v1.11.0, run the following:

```
minikube start --kubernetes-version=v1.11.0 --cpus 4
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
    "difficulty": "0x400",                                                      
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

```bash
bazel run //k8s:everything.apply
```

This creates a few nodes and one miner with CPU restrictions. After ~30 
minutes, the miner has generated the DAG and begins mining. This seems iterate
over 2 DAG epochs dispite the flags set for 1 DAG in memory and in disk. 

Note: This can be improved by giving the miner more CPU.

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