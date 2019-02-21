# Cluster private key management tool

This is a primative tool for managing and delegating validator private key 
assigments within the kubernetes cluster.

## Design

When a validator pod is initializing within the cluster, it requests a private 
key for a deposited validator. Since pods are epheremal, scale up/down quickly,
there needs to be some service to manage private key allocations, validator 
deposits, and re-allocations of previously in-use private keys from terminated
pods. 

Workflow for bootstraping a validator pod

1. Request a private key from the pk manager.
1. If an unallocated private key exists (from previously terminated pod), assign
   to the requesting pod.
1. If all available private keys are in use, generate a new private key, and
   make the deposit on behalf of this newly generated private key.
1. Write the assignment to some persistent datastore and fulfill the request.
1. The validator uses this private key to act as a deposited validator in the
   system. 

## Server

The server manages the private key database, allocates new private keys, makes
validator deposits, and fulfills requests from pods for private key allocation.

## Client

The client makes the private key request with a given pod name and generates a 
keystore with the server response.
