# Prysm Client Interoperability Guide

This README details how to setup Prysm  for interop testing for usage with other Ethereum 2.0 clients.

## Installation & Setup

1. Install Bazel

## Starting from Genesis

Prysm supports a few ways to quickly launch a beacon node from basic configurations:

- [`Yaml Genesis`](#yaml-genesis): Launches a beacon node from a .yaml file containing a genesis beacon state **(recommended)**
- [`SSZ Genesis`](#ssz-genesis): Launches a beacon node  from a .ssz file containing a SSZ-encoded, genesis beacon state

## Interop File Setups

To setup the necessary files for these quick starts, Prysm provides tools to generate `genesis.yaml` or `genesis.ssz` from an
encrypted, Prysm validator keystore, an unencrypted yaml of validator keys following the official interop YAML format 
[here](https://github.com/ethereum/eth2.0-pm/blob/master/interop/mocked_start/keygen_10000_validators.yaml), or an unencrypted JSON file of validator keys.

The following options are available:

#### From an Encrypted, Prysm Validator Keystore
```
```

#### From an Unencrypted, YAML File of Validator Keys
```
```

#### From an Unencrypted, JSON File of Validator Keys
```
```

## Launching a Beacon Node + Validator Client

###  Yaml Genesis

### SSZ Genesis
