# Prysm Client Interoperability Guide

This README details how to setup Prysm  for interop testing for usage with other Ethereum 2.0 clients.

## Installation & Setup

1. Install Bazel

## Starting from Genesis

Prysm supports a few ways to quickly launch a beacon node from basic configurations:

- [`Yaml Genesis`](#yaml-genesis): Launches a beacon node from a .yaml file specifying a genesis beacon state **(recommended)**
- [`SSZ Genesis`](#ssz-genesis): Launches a beacon node  from a .ssz file containing a SSZ-encoded, genesis beacon state
- [`JSON Genesis`](#json-genesis): Launches a beacon node  from a .json file specifying a genesis beacon state

## Launching a Beacon Node + Validator Client

To setup the necessary files for these quick starts, Prysm provides a tool to generate `genesis.yaml`, `genesis.ssz`, `genesis.json` from an
a deterministically generated set of validator private keys following the official interop YAML format 
[here](https://github.com/ethereum/eth2.0-pm/blob/master/interop/mocked_start).

###  Yaml Genesis

### SSZ Genesis
