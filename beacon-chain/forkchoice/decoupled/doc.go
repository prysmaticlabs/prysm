// Package decoupled implements Goldilocks fork choice. Finality gadget (FG)
// state restricts the eligible branches; available-chain (AC) votes select the head.
// F and J denote the store's accepted finalized and justified blocks.
//
// Callers validate inputs and serialize access to Store, including reads.
// Returned nodes are store-owned and must not be modified.
//
// Ticks precede messages at the same timestamp. Callers must replay missed phases
// in order: OnTick processes only the slot containing its argument. Read methods
// do not advance ticks.
package decoupled
