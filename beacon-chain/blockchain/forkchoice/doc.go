/*
Package forkchoice implements the Latest Message Driven GHOST (Greediest Heaviest Observed
Sub-Tree) algorithm as the Ethereum Serenity beacon chain fork choice rule. This algorithm is designed to
properly detect the canonical chain based on validator votes even in the presence of high network
latency, network partitions, and many conflicting blocks. To read more about fork choice, read the
official accompanying document:
https://github.com/ethereum/eth2.0-specs/blob/v0.8.3/specs/core/0_fork-choice.md
*/
package forkchoice
