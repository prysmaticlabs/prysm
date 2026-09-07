### Fixed

- Return the confirmed block's payload hash as the safe execution block hash even when it is zero (pre-merge, genesis in spec tests) instead of falling back to the unrealized justified hash; the fallback now only covers a confirmed root pruned from forkchoice.
