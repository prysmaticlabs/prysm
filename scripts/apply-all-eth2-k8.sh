#! /bin/bash

kubectl apply -f

kubectl apply -f k8s/beacon-chain/namespace.yaml
kubectl apply -f k8s/priority.yaml
kubectl apply -f k8s/beacon-chain/beacon-config.config.yaml
kubectl apply -f k8s/beacon-chain/beacon-chain.deploy.yaml
kubectl apply -f k8s/beacon-chain/beacon-chain.service.yaml

kubectl apply -f k8s/beacon-chain/cluster-manager.encrypted_secret.yaml
kubectl apply -f k8s/beacon-chain/cluster-manager.yaml
kubectl apply -f k8s/beacon-chain/validator.deploy.yaml
