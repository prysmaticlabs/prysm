
<h1 align="center">
  <a href="libp2p.io"><img width="250" src="https://github.com/libp2p/libp2p/blob/master/logo/black-bg-2.png?raw=true" alt="libp2p hex logo" /></a>
</h1>

<h3 align="center">The Go implementation of the libp2p Networking Stack.</h3>

<p align="center">
  <a href="http://ipn.io"><img src="https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg?style=flat-square" /></a>
  <a href="http://libp2p.io/"><img src="https://img.shields.io/badge/project-libp2p-blue.svg?style=flat-square" /></a>
  <a href="http://webchat.freenode.net/?channels=%23ipfs"><img src="https://img.shields.io/badge/freenode-%23ipfs-blue.svg?style=flat-square" /></a>
  <a href="https://waffle.io/libp2p/libp2p"><img src="https://img.shields.io/badge/pm-waffle-blue.svg?style=flat-square" /></a>
</p>

<p align="center">
  <a href="https://travis-ci.org/libp2p/go-libp2p"><img src="https://travis-ci.org/libp2p/go-libp2p.svg?branch=master" /></a>
  <br>
  <a href="https://github.com/RichardLitt/standard-readme"><img src="https://img.shields.io/badge/standard--readme-OK-green.svg?style=flat-square" /></a>
  <a href="https://godoc.org/github.com/libp2p/go-libp2p"><img src="https://godoc.org/github.com/ipfs/go-libp2p?status.svg" /></a>
  <a href=""><img src="https://img.shields.io/badge/golang-%3E%3D1.8.0-orange.svg?style=flat-square" /></a>
  <br>
</p>

# Project status

[![Throughput Graph](https://graphs.waffle.io/libp2p/go-libp2p/throughput.svg)](https://waffle.io/libp2p/go-libp2p/metrics/throughput)

[**`Weekly Core Dev Calls`**](https://github.com/ipfs/pm/issues/674)

# Table of Contents

- [Background](#background)
- [Bundles](#bundles)
- [Usage](#usage)
  - [Install](#install)
  - [API](#api)
  - [Examples](#examples)
- [Development](#development)
  - [Tests](#tests)
  - [Packages](#packages)
- [Contribute](#contribute)
- [License](#license)

## Background

[libp2p](https://github.com/libp2p/specs) is a networking stack and library modularized out of [The IPFS Project](https://github.com/ipfs/ipfs), and bundled separately for other tools to use.
>
libp2p is the product of a long, and arduous quest of understanding -- a deep dive into the internet's network stack, and plentiful peer-to-peer protocols from the past. Building large scale peer-to-peer systems has been complex and difficult in the last 15 years, and libp2p is a way to fix that. It is a "network stack" -- a protocol suite -- that cleanly separates concerns, and enables sophisticated applications to only use the protocols they absolutely need, without giving up interoperability and upgradeability. libp2p grew out of IPFS, but it is built so that lots of people can use it, for lots of different projects.
>
> We will be writing a set of docs, posts, tutorials, and talks to explain what p2p is, why it is tremendously useful, and how it can help your existing and new projects. But in the meantime, check out
>
> - [**The libp2p Specification**](https://github.com/libp2p/specs)
> - [**go-libp2p implementation**](https://github.com/libp2p/go-libp2p)
> - [**js-libp2p implementation**](https://github.com/libp2p/js-libp2p)


## Bundles

There is currently only one bundle of `go-libp2p`, this package. This bundle is used by [`go-ipfs`](https://github.com/ipfs/go-ipfs).

## Usage

`go-libp2p` repo is a place holder for the list of Go modules that compose Go libp2p, as well as its entry point.

### Install

```bash
> go get -u -d github.com/libp2p/go-libp2p/...
> cd $GOPATH/src/github.com/libp2p/go-libp2p
> make
> make deps
```

### API

[![GoDoc](https://godoc.org/github.com/ipfs/go-libp2p?status.svg)](https://godoc.org/github.com/libp2p/go-libp2p)

### Examples

Examples can be found in the [examples repo](https://github.com/libp2p/go-libp2p-examples).

## Development

### Dependencies

While developing, you need to use [gx to install and link your dependencies](https://github.com/whyrusleeping/gx#dependencies), to do that, run:

```sh
> make deps
```

Before commiting and pushing to Github, make sure to rewind the gx'ify of dependencies. You can do that with:

```sh
> make publish
```

### Tests

Running of individual tests is done through `gx test <path to test>`

```bash
$ cd $GOPATH/src/github.com/libp2p/go-libp2p
$ make deps
$ gx test ./p2p/<path of module you want to run tests for>
```

### Packages

> This table is generated using the module [`package-table`](https://github.com/ipfs-shipyard/package-table) with `package-table --data=package-list.json`.

List of packages currently in existence for libp2p:

| Name | CI/Travis | CI/Jenkins | Coverage | Description |
| ---------|---------|---------|---------|--------- |
| **Libp2p** |
| [`go-libp2p`](//github.com/libp2p/go-libp2p) | [![Travis CI](https://travis-ci.org/libp2p/go-libp2p.svg?branch=master)](https://travis-ci.org/libp2p/go-libp2p) | [![jenkins](https://ci.ipfs.team/buildStatus/icon?job=libp2p/go-libp2p/master)](https://ci.ipfs.team/job/libp2p/job/go-libp2p/job/master/) | [![codecov](https://codecov.io/gh/libp2p/go-libp2p/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-libp2p) | go-libp2p entry point |
| [`go-libp2p-host`](//github.com/libp2p/go-libp2p-host) | [![Travis CI](https://travis-ci.org/libp2p/go-libp2p-host.svg?branch=master)](https://travis-ci.org/libp2p/go-libp2p-host) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-libp2p-host/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-libp2p-host) | libp2p "host" interface |
| [`go-libp2p-blankhost`](//github.com/libp2p/go-libp2p-blankhost) | [![Travis CI](https://travis-ci.org/libp2p/go-libp2p-blankhost.svg?branch=master)](https://travis-ci.org/libp2p/go-libp2p-blankhost) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-libp2p-blankhost/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-libp2p-blankhost) | minimal implementation of the "host" interface |
| **Network** |
| [`go-libp2p-net`](//github.com/libp2p/go-libp2p-net) | [![Travis CI](https://travis-ci.org/libp2p/go-libp2p-net.svg?branch=master)](https://travis-ci.org/libp2p/go-libp2p-net) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-libp2p-net/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-libp2p-net) | libp2p connection and "network" interfaces |
| [`go-libp2p-swarm`](//github.com/libp2p/go-libp2p-swarm) | [![Travis CI](https://travis-ci.org/libp2p/go-libp2p-swarm.svg?branch=master)](https://travis-ci.org/libp2p/go-libp2p-swarm) | [![jenkins](https://ci.ipfs.team/buildStatus/icon?job=libp2p/go-libp2p-swarm/master)](https://ci.ipfs.team/job/libp2p/job/go-libp2p-swarm/job/master/) | [![codecov](https://codecov.io/gh/libp2p/go-libp2p-swarm/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-libp2p-swarm) | reference implementation |
| **Transport** |
| [`go-libp2p-transport`](//github.com/libp2p/go-libp2p-transport) | [![Travis CI](https://travis-ci.org/libp2p/go-libp2p-transport.svg?branch=master)](https://travis-ci.org/libp2p/go-libp2p-transport) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-libp2p-transport/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-libp2p-transport) | interfaces |
| [`go-ws-transport`](//github.com/libp2p/go-ws-transport) | [![Travis CI](https://travis-ci.org/libp2p/go-ws-transport.svg?branch=master)](https://travis-ci.org/libp2p/go-ws-transport) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-ws-transport/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-ws-transport) | WebSocket transport |
| [`go-tcp-transport`](//github.com/libp2p/go-tcp-transport) | [![Travis CI](https://travis-ci.org/libp2p/go-tcp-transport.svg?branch=master)](https://travis-ci.org/libp2p/go-tcp-transport) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-tcp-transport/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-tcp-transport) | TCP transport |
| [`go-libp2p-quic-transport`](//github.com/libp2p/go-libp2p-quic-transport) | [![Travis CI](https://travis-ci.org/libp2p/go-libp2p-quic-transport.svg?branch=master)](https://travis-ci.org/libp2p/go-libp2p-quic-transport) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-libp2p-quic-transport/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-libp2p-quic-transport) | QUIC transport |
| [`go-udp-transport`](//github.com/libp2p/go-udp-transport) | [![Travis CI](https://travis-ci.org/libp2p/go-udp-transport.svg?branch=master)](https://travis-ci.org/libp2p/go-udp-transport) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-udp-transport/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-udp-transport) | UDP transport |
| [`go-utp-transport`](//github.com/libp2p/go-utp-transport) | [![Travis CI](https://travis-ci.org/libp2p/go-utp-transport.svg?branch=master)](https://travis-ci.org/libp2p/go-utp-transport) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-utp-transport/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-utp-transport) | uTorrent transport (UTP) |
| [`go-libp2p-circuit`](//github.com/libp2p/go-libp2p-circuit) | [![Travis CI](https://travis-ci.org/libp2p/go-libp2p-circuit.svg?branch=master)](https://travis-ci.org/libp2p/go-libp2p-circuit) | [![jenkins](https://ci.ipfs.team/buildStatus/icon?job=libp2p/go-libp2p-circuit/master)](https://ci.ipfs.team/job/libp2p/job/go-libp2p-circuit/job/master/) | [![codecov](https://codecov.io/gh/libp2p/go-libp2p-circuit/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-libp2p-circuit) | relay transport |
| [`go-libp2p-transport-upgrader`](//github.com/libp2p/go-libp2p-transport-upgrader) | [![Travis CI](https://travis-ci.org/libp2p/go-libp2p-transport-upgrader.svg?branch=master)](https://travis-ci.org/libp2p/go-libp2p-transport-upgrader) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-libp2p-transport-upgrader/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-libp2p-transport-upgrader) | upgrades multiaddr-net connections into full libp2p transports |
| [`go-libp2p-reuseport-transport`](//github.com/libp2p/go-libp2p-reuseport-transport) | [![Travis CI](https://travis-ci.org/libp2p/go-libp2p-reuseport-transport.svg?branch=master)](https://travis-ci.org/libp2p/go-libp2p-reuseport-transport) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-libp2p-reuseport-transport/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-libp2p-reuseport-transport) | partial transport for building transports that reuse ports |
| **Encrypted Channels** |
| [`go-conn-security`](//github.com/libp2p/go-conn-security) | [![Travis CI](https://travis-ci.org/libp2p/go-conn-security.svg?branch=master)](https://travis-ci.org/libp2p/go-conn-security) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-conn-security/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-conn-security) | interfaces |
| [`go-libp2p-secio`](//github.com/libp2p/go-libp2p-secio) | [![Travis CI](https://travis-ci.org/libp2p/go-libp2p-secio.svg?branch=master)](https://travis-ci.org/libp2p/go-libp2p-secio) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-libp2p-secio/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-libp2p-secio) | SecIO crypto channel |
| [`go-conn-security-multistream`](//github.com/libp2p/go-conn-security-multistream) | [![Travis CI](https://travis-ci.org/libp2p/go-conn-security-multistream.svg?branch=master)](https://travis-ci.org/libp2p/go-conn-security-multistream) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-conn-security-multistream/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-conn-security-multistream) | multistream multiplexed meta crypto channel |
| **Private Network** |
| [`go-libp2p-interface-pnet`](//github.com/libp2p/go-libp2p-interface-pnet) | [![Travis CI](https://travis-ci.org/libp2p/go-libp2p-interface-pnet.svg?branch=master)](https://travis-ci.org/libp2p/go-libp2p-interface-pnet) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-libp2p-interface-pnet/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-libp2p-interface-pnet) | interfaces |
| [`go-libp2p-pnet`](//github.com/libp2p/go-libp2p-pnet) | [![Travis CI](https://travis-ci.org/libp2p/go-libp2p-pnet.svg?branch=master)](https://travis-ci.org/libp2p/go-libp2p-pnet) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-libp2p-pnet/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-libp2p-pnet) | reference implementation |
| **Stream Muxers** |
| [`go-stream-muxer`](//github.com/libp2p/go-stream-muxer) | [![Travis CI](https://travis-ci.org/libp2p/go-stream-muxer.svg?branch=master)](https://travis-ci.org/libp2p/go-stream-muxer) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-stream-muxer/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-stream-muxer) | interfaces |
| [`go-smux-yamux`](//github.com/whyrusleeping/go-smux-yamux) | [![Travis CI](https://travis-ci.org/whyrusleeping/go-smux-yamux.svg?branch=master)](https://travis-ci.org/whyrusleeping/go-smux-yamux) | N/A | [![codecov](https://codecov.io/gh/whyrusleeping/go-smux-yamux/branch/master/graph/badge.svg)](https://codecov.io/gh/whyrusleeping/go-smux-yamux) | YAMUX stream multiplexer |
| [`go-smux-mplex`](//github.com/whyrusleeping/go-smux-mplex) | [![Travis CI](https://travis-ci.org/whyrusleeping/go-smux-mplex.svg?branch=master)](https://travis-ci.org/whyrusleeping/go-smux-mplex) | N/A | [![codecov](https://codecov.io/gh/whyrusleeping/go-smux-mplex/branch/master/graph/badge.svg)](https://codecov.io/gh/whyrusleeping/go-smux-mplex) | MPLEX stream multiplexer |
| **NAT Traversal** |
| [`go-libp2p-nat`](//github.com/libp2p/go-libp2p-nat) | [![Travis CI](https://travis-ci.org/libp2p/go-libp2p-nat.svg?branch=master)](https://travis-ci.org/libp2p/go-libp2p-nat) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-libp2p-nat/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-libp2p-nat) |  |
| **Peerstore** |
| [`go-libp2p-peerstore`](//github.com/libp2p/go-libp2p-peerstore) | [![Travis CI](https://travis-ci.org/libp2p/go-libp2p-peerstore.svg?branch=master)](https://travis-ci.org/libp2p/go-libp2p-peerstore) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-libp2p-peerstore/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-libp2p-peerstore) | interfaces and reference implementation |
| **Connection Manager** |
| [`go-libp2p-interface-connmgr`](//github.com/libp2p/go-libp2p-interface-connmgr) | [![Travis CI](https://travis-ci.org/libp2p/go-libp2p-interface-connmgr.svg?branch=master)](https://travis-ci.org/libp2p/go-libp2p-interface-connmgr) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-libp2p-interface-connmgr/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-libp2p-interface-connmgr) | interface |
| [`go-libp2p-connmgr`](//github.com/libp2p/go-libp2p-connmgr) | [![Travis CI](https://travis-ci.org/libp2p/go-libp2p-connmgr.svg?branch=master)](https://travis-ci.org/libp2p/go-libp2p-connmgr) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-libp2p-connmgr/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-libp2p-connmgr) | reference implementation |
| **Routing** |
| [`go-libp2p-routing`](//github.com/libp2p/go-libp2p-routing) | [![Travis CI](https://travis-ci.org/libp2p/go-libp2p-routing.svg?branch=master)](https://travis-ci.org/libp2p/go-libp2p-routing) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-libp2p-routing/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-libp2p-routing) | routing interfaces |
| [`go-libp2p-record`](//github.com/libp2p/go-libp2p-record) | [![Travis CI](https://travis-ci.org/libp2p/go-libp2p-record.svg?branch=master)](https://travis-ci.org/libp2p/go-libp2p-record) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-libp2p-record/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-libp2p-record) | record type and validator logic |
| [`go-libp2p-routing-helpers`](//github.com/libp2p/go-libp2p-routing-helpers) | [![Travis CI](https://travis-ci.org/libp2p/go-libp2p-routing-helpers.svg?branch=master)](https://travis-ci.org/libp2p/go-libp2p-routing-helpers) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-libp2p-routing-helpers/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-libp2p-routing-helpers) | helpers for composing routers |
| [`go-libp2p-kad-dht`](//github.com/libp2p/go-libp2p-kad-dht) | [![Travis CI](https://travis-ci.org/libp2p/go-libp2p-kad-dht.svg?branch=master)](https://travis-ci.org/libp2p/go-libp2p-kad-dht) | [![jenkins](https://ci.ipfs.team/buildStatus/icon?job=libp2p/go-libp2p-kad-dht/master)](https://ci.ipfs.team/job/libp2p/job/go-libp2p-kad-dht/job/master/) | [![codecov](https://codecov.io/gh/libp2p/go-libp2p-kad-dht/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-libp2p-kad-dht) | Kademlia-like router |
| [`go-libp2p-pubsub-router`](//github.com/libp2p/go-libp2p-pubsub-router) | [![Travis CI](https://travis-ci.org/libp2p/go-libp2p-pubsub-router.svg?branch=master)](https://travis-ci.org/libp2p/go-libp2p-pubsub-router) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-libp2p-pubsub-router/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-libp2p-pubsub-router) | record-store over pubsub adapter |
| **Consensus** |
| [`go-libp2p-consensus`](//github.com/libp2p/go-libp2p-consensus) | [![Travis CI](https://travis-ci.org/libp2p/go-libp2p-consensus.svg?branch=master)](https://travis-ci.org/libp2p/go-libp2p-consensus) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-libp2p-consensus/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-libp2p-consensus) | consensus protocols interfaces |
| [`go-libp2p-raft`](//github.com/libp2p/go-libp2p-raft) | [![Travis CI](https://travis-ci.org/libp2p/go-libp2p-raft.svg?branch=master)](https://travis-ci.org/libp2p/go-libp2p-raft) | [![jenkins](https://ci.ipfs.team/buildStatus/icon?job=libp2p/go-libp2p-raft/master)](https://ci.ipfs.team/job/libp2p/job/go-libp2p-raft/job/master/) | [![codecov](https://codecov.io/gh/libp2p/go-libp2p-raft/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-libp2p-raft) | consensus implementation over raft |
| **Pubsub** |
| [`go-floodsub`](//github.com/libp2p/go-floodsub) | [![Travis CI](https://travis-ci.org/libp2p/go-floodsub.svg?branch=master)](https://travis-ci.org/libp2p/go-floodsub) | [![jenkins](https://ci.ipfs.team/buildStatus/icon?job=libp2p/go-floodsub/master)](https://ci.ipfs.team/job/libp2p/job/go-floodsub/job/master/) | [![codecov](https://codecov.io/gh/libp2p/go-floodsub/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-floodsub) | pubsub implementation |
| **Metrics** |
| [`go-libp2p-metrics`](//github.com/libp2p/go-libp2p-metrics) | [![Travis CI](https://travis-ci.org/libp2p/go-libp2p-metrics.svg?branch=master)](https://travis-ci.org/libp2p/go-libp2p-metrics) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-libp2p-metrics/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-libp2p-metrics) | libp2p metrics interfaces/collectors |
| **Data Types** |
| [`go-libp2p-peer`](//github.com/libp2p/go-libp2p-peer) | [![Travis CI](https://travis-ci.org/libp2p/go-libp2p-peer.svg?branch=master)](https://travis-ci.org/libp2p/go-libp2p-peer) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-libp2p-peer/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-libp2p-peer) | libp2p peer-ID datatype |
| [`go-libp2p-crypto`](//github.com/libp2p/go-libp2p-crypto) | [![Travis CI](https://travis-ci.org/libp2p/go-libp2p-crypto.svg?branch=master)](https://travis-ci.org/libp2p/go-libp2p-crypto) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-libp2p-crypto/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-libp2p-crypto) | libp2p key types |
| [`go-libp2p-protocol`](//github.com/libp2p/go-libp2p-protocol) | [![Travis CI](https://travis-ci.org/libp2p/go-libp2p-protocol.svg?branch=master)](https://travis-ci.org/libp2p/go-libp2p-protocol) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-libp2p-protocol/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-libp2p-protocol) | libp2p protocol datatype |
| [`go-libp2p-kbucket`](//github.com/libp2p/go-libp2p-kbucket) | [![Travis CI](https://travis-ci.org/libp2p/go-libp2p-kbucket.svg?branch=master)](https://travis-ci.org/libp2p/go-libp2p-kbucket) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-libp2p-kbucket/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-libp2p-kbucket) | Kademlia routing table helper types |
| **Utilities/miscellaneous** |
| [`go-libp2p-loggables`](//github.com/libp2p/go-libp2p-loggables) | [![Travis CI](https://travis-ci.org/libp2p/go-libp2p-loggables.svg?branch=master)](https://travis-ci.org/libp2p/go-libp2p-loggables) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-libp2p-loggables/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-libp2p-loggables) | logging helpers |
| [`go-maddr-filter`](//github.com/libp2p/go-maddr-filter) | [![Travis CI](https://travis-ci.org/libp2p/go-maddr-filter.svg?branch=master)](https://travis-ci.org/libp2p/go-maddr-filter) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-maddr-filter/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-maddr-filter) | multiaddr filtering helpers |
| [`go-libp2p-netutil`](//github.com/libp2p/go-libp2p-netutil) | [![Travis CI](https://travis-ci.org/libp2p/go-libp2p-netutil.svg?branch=master)](https://travis-ci.org/libp2p/go-libp2p-netutil) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-libp2p-netutil/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-libp2p-netutil) | misc utilities |
| [`go-msgio`](//github.com/libp2p/go-msgio) | [![Travis CI](https://travis-ci.org/libp2p/go-msgio.svg?branch=master)](https://travis-ci.org/libp2p/go-msgio) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-msgio/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-msgio) | length prefixed data channel |
| [`go-addr-util`](//github.com/libp2p/go-addr-util) | [![Travis CI](https://travis-ci.org/libp2p/go-addr-util.svg?branch=master)](https://travis-ci.org/libp2p/go-addr-util) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-addr-util/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-addr-util) | address utilities for libp2p swarm |
| [`go-buffer-pool`](//github.com/libp2p/go-buffer-pool) | [![Travis CI](https://travis-ci.org/libp2p/go-buffer-pool.svg?branch=master)](https://travis-ci.org/libp2p/go-buffer-pool) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-buffer-pool/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-buffer-pool) | a variable size buffer pool for go |
| [`go-libp2p-loggables`](//github.com/libp2p/go-libp2p-loggables) | [![Travis CI](https://travis-ci.org/libp2p/go-libp2p-loggables.svg?branch=master)](https://travis-ci.org/libp2p/go-libp2p-loggables) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-libp2p-loggables/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-libp2p-loggables) | logging helpers |
| [`go-libp2p-routing-helpers`](//github.com/libp2p/go-libp2p-routing-helpers) | [![Travis CI](https://travis-ci.org/libp2p/go-libp2p-routing-helpers.svg?branch=master)](https://travis-ci.org/libp2p/go-libp2p-routing-helpers) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-libp2p-routing-helpers/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-libp2p-routing-helpers) | routing helpers |
| [`go-maddr-filter`](//github.com/libp2p/go-maddr-filter) | [![Travis CI](https://travis-ci.org/libp2p/go-maddr-filter.svg?branch=master)](https://travis-ci.org/libp2p/go-maddr-filter) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-maddr-filter/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-maddr-filter) | a library to perform filtering of multiaddrs. |
| [`go-reuseport`](//github.com/libp2p/go-reuseport) | [![Travis CI](https://travis-ci.org/libp2p/go-reuseport.svg?branch=master)](https://travis-ci.org/libp2p/go-reuseport) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-reuseport/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-reuseport) | enables reuse of addresses |
| [`go-sockaddr`](//github.com/libp2p/go-sockaddr) | [![Travis CI](https://travis-ci.org/libp2p/go-sockaddr.svg?branch=master)](https://travis-ci.org/libp2p/go-sockaddr) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-sockaddr/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-sockaddr) | utils for sockaddr conversions |
| [`go-flow-metrics`](//github.com/libp2p/go-flow-metrics) | [![Travis CI](https://travis-ci.org/libp2p/go-flow-metrics.svg?branch=master)](https://travis-ci.org/libp2p/go-flow-metrics) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-flow-metrics/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-flow-metrics) | metrics library |
| **Testing and examples** |
| [`go-testutil`](//github.com/libp2p/go-testutil) | [![Travis CI](https://travis-ci.org/libp2p/go-testutil.svg?branch=master)](https://travis-ci.org/libp2p/go-testutil) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-testutil/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-testutil) | a collection of testing utilities for ipfs and libp2p |
| [`go-libp2p-examples`](//github.com/libp2p/go-libp2p-examples) | [![Travis CI](https://travis-ci.org/libp2p/go-libp2p-examples.svg?branch=master)](https://travis-ci.org/libp2p/go-libp2p-examples) | [![jenkins](https://ci.ipfs.team/buildStatus/icon?job=libp2p/go-libp2p-examples/master)](https://ci.ipfs.team/job/libp2p/job/go-libp2p-examples/job/master/) | [![codecov](https://codecov.io/gh/libp2p/go-libp2p-examples/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-libp2p-examples) | go-libp2p examples and tutorials |
| [`go-libp2p-circuit-progs`](//github.com/libp2p/go-libp2p-circuit-progs) | [![Travis CI](https://travis-ci.org/libp2p/go-libp2p-circuit-progs.svg?branch=master)](https://travis-ci.org/libp2p/go-libp2p-circuit-progs) | N/A | [![codecov](https://codecov.io/gh/libp2p/go-libp2p-circuit-progs/branch/master/graph/badge.svg)](https://codecov.io/gh/libp2p/go-libp2p-circuit-progs) | testing programs for go-libp2p-circuit |

# Contribute

go-libp2p is part of [The IPFS Project](https://github.com/ipfs/ipfs), and is MIT licensed open source software. We welcome contributions big and small! Take a look at the [community contributing notes](https://github.com/ipfs/community/blob/master/contributing.md). Please make sure to check the [issues](https://github.com/ipfs/go-libp2p/issues). Search the closed ones before reporting things, and help us with the open ones.

Guidelines:

- read the [libp2p spec](https://github.com/libp2p/specs)
- please make branches + pull-request, even if working on the main repository
- ask questions or talk about things in [Issues](https://github.com/libp2p/go-libp2p/issues) or #ipfs on freenode.
- ensure you are able to contribute (no legal issues please-- we use the DCO)
- run `go fmt` before pushing any code
- run `golint` and `go vet` too -- some things (like protobuf files) are expected to fail.
- get in touch with @jbenet and @diasdavid about how best to contribute
- have fun!

There's a few things you can do right now to help out:
 - Go through the modules below and **check out existing issues**. This would be especially useful for modules in active development. Some knowledge of IPFS/libp2p may be required, as well as the infrasture behind it - for instance, you may need to read up on p2p and more complex operations like muxing to be able to help technically.
 - **Perform code reviews**.
 - **Add tests**. There can never be enough tests.

## Modularizing go-libp2p

We have currently a work in progress of modularizing go-libp2p from a repo monolith to several packages in different repos that can be reused for other projects of swapped for custom libp2p builds.

We want to maintain history, so we'll use git-subtree for extracting packages. Find instructions below:

```sh
# 1) create the extracted tree (has the directory specified as -P as its root)
> cd go-libp2p/
> git subtree split -P p2p/crypto/secio/ -b libp2p-secio
62b0a5c21574bcbe06c422785cd5feff378ae5bd
# important to delete the tree now, so that outdated imports fail in step 5
> git rm -r p2p/crypto/secio/
> git commit
> cd ../

# 2) make the new repo
> mkdir go-libp2p-secio
> cd go-libp2p-secio/
> git init && git commit --allow-empty

# 3) fetch the extracted tree from the previous repo
> git remote add libp2p ../go-libp2p
> git fetch libp2p
> git reset --hard libp2p/libp2p-secio

# 4) update self import paths
> sed -someflagsidontknow 'go-libp2p/p2p/crypto/secio' 'golibp2p-secio'
> git commit

# 5) create package.json and check all imports are correct
> vim package.json
> gx --verbose install --global
> gx-go rewrite
> go test ./...
> gx-go rewrite --undo
> git commit

# 4) make the package ready
> vim README.md LICENSE
> git commit

# 5) bump the version separately
> vim package.json
> gx publish
> git add package.json .gx/
> git commit -m 'Publish 1.2.3'

# 6) clean up and push
> git remote rm libp2p
> git push origin master
```
