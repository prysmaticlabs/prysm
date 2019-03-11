# Minikube setup for Beacon-chain and 8 Validators

## Set up Kubernetes and Minikube

### Make sure you have kubectl installed

https://kubernetes.io/docs/tasks/tools/install-minikube/

### For installing the hypervisor, it's recommended if you're on OS X to use hyperkit, install it here

https://github.com/kubernetes/minikube/blob/master/docs/drivers.md#hyperkit-driver

### Configure Hyperkit as your default

`minikube config set vm-driver hyperkit`

## Running the beacon chain and validators in minkube

### Start Minikube and the dashboard

`minikube start --memory 4096`

`minikube dashboard`

### Apply the namespace and config yamls

```
cd k8s/
kubectl apply -f priority.yaml

cd beacon-chain
kubectl apply -f namespace.yaml
kubectl apply -f beacon-config.config.yaml
```

### Apply the beacon chain yamls to start the beacon-chain

```
kubectl apply -f beacon-chain.service.yaml
kubectl apply -f beacon-chain.deploy.yaml
```

### Go into the minikube dashboard and set your namespace to "beacon-chain" in the middle of the left side

You should see 3 beacon-chain node replicas

### Apply the cluster manager and validator yamls

```
kubectl apply -f cluster-manager.encrypted_secret.yaml
kubectl apply -f cluster-manager.yaml
kubectl apply -f validator.deploy.yaml
```

### Check the beacon-chain namespace in the dashboard and you should see 8 validators loading

# Minikube common commands

## To remove all the containers inside the minikube

`kubectl delete -f beacon-chain

## To stop the minikube instance

`minikube stop`

## To delete the container

`minikube delete`
