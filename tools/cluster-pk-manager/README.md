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

1. Request `n` private keys from the pk manager.
1. If unallocated private keys exist (from previously terminated pods), assign
   to the requesting pod.
1. If there are not at least `n` keys not in use, generate new private keys, 
   and make the deposits on behalf of these newly generated private keys.
1. Write the key allocations to a persistent datastore and fulfill the request.
1. The client uses these private keys to act as deposited validators in the
   system. 

## Server

The server manages the private key database, allocates new private keys, makes
validator deposits, and fulfills requests from pods for private key allocation.

### Database structure

There are two buckets for the server, unallocated keys and allocated keys.

Unallocated keys bucket:

| key         | value |
|-------------|-------|
| private key | nil   |

Allocated keys bucket:

| key      | value                |
|----------|----------------------|
| pod name | list of private keys |

### Key management design

There are two types of operations with regards to private keys: 

- Allocate(podName, keys)
- UnallocateAllKeys(podName)

Allocating keys will first check and attempt to recycle existing, unused keys.
If there are no unused keys available (or not enough), new keys are deposited.

Unallocating keys happens when a pod is destroyed. This should return all of 
that's pods' keys to the unallocated keys bucket.

### Assignments HTTP Page `/assignments`

The server exposes an HTTP page which maps pod names to public keys.
This may be useful for determining which logs to follow for a given validator.

## Client

The client makes the private key request with a given pod name and generates a 
keystore with the server response.
