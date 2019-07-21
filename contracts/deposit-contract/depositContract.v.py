## compiled with v0.1.0-beta.9 ##
DEPOSIT_CONTRACT_TREE_DEPTH: constant(uint256) = 32
SECONDS_PER_DAY: constant(uint256) = 86400
MAX_DEPOSIT_COUNT: constant(uint256) = 4294967295 # 2**DEPOSIT_CONTRACT_TREE_DEPTH - 1
PUBKEY_LENGTH: constant(uint256) = 48  # bytes
WITHDRAWAL_CREDENTIALS_LENGTH: constant(uint256) = 32  # bytes
AMOUNT_LENGTH: constant(uint256) = 8  # bytes
SIGNATURE_LENGTH: constant(uint256) = 96  # bytes

DepositEvent: event({
    pubkey: bytes[48],
    withdrawal_credentials: bytes[32],
    amount: bytes[8],
    signature: bytes[96],
    index: bytes[8],
})

MIN_DEPOSIT_AMOUNT: public(uint256) # Gwei
zero_hashes: bytes32[DEPOSIT_CONTRACT_TREE_DEPTH]
branch: bytes32[DEPOSIT_CONTRACT_TREE_DEPTH]
deposit_count: public(uint256)
drain_address: public(address)

@public
def __init__( # Parameters for debugging, not for production use!
        minDeposit: uint256,
        _drain_address: address):
    self.MIN_DEPOSIT_AMOUNT = minDeposit
    self.drain_address = _drain_address
    for i in range(DEPOSIT_CONTRACT_TREE_DEPTH - 1):
        self.zero_hashes[i+1] = sha256(concat(self.zero_hashes[i], self.zero_hashes[i]))

@private
@constant
def to_bytes8(value: uint256) -> bytes[8]:
    return slice(convert(value, bytes32), start=24, len=8)


@private
@constant
def to_little_endian_64(value: uint256) -> bytes[8]:
    # Reversing bytes using bitwise uint256 manipulations
    # Note: array accesses of bytes[] are not currently supported in Vyper
    # Note: this function is only called when `value < 2**64`
    y: uint256 = 0
    x: uint256 = value
    for _ in range(8):
        y = shift(y, 8)
        y = y + bitwise_and(x, 255)
        x = shift(x, -8)
    return slice(convert(y, bytes32), start=24, len=8)


@public
@constant
def get_hash_tree_root() -> bytes32:
    zero_bytes32: bytes32 = 0x0000000000000000000000000000000000000000000000000000000000000000
    node: bytes32 = zero_bytes32
    size: uint256 = self.deposit_count
    for height in range(DEPOSIT_CONTRACT_TREE_DEPTH):
        if bitwise_and(size, 1) == 1:  # More gas efficient than `size % 2 == 1`
            node = sha256(concat(self.branch[height], node))
        else:
            node = sha256(concat(node, self.zero_hashes[height]))
        size /= 2
    return sha256(concat(node, self.to_little_endian_64(self.deposit_count), slice(zero_bytes32, start=0, len=24)))


@public
@constant
def get_deposit_count() -> bytes[8]:
    return self.to_little_endian_64(self.deposit_count)


@payable
@public
def deposit(pubkey: bytes[PUBKEY_LENGTH],
            withdrawal_credentials: bytes[WITHDRAWAL_CREDENTIALS_LENGTH],
            signature: bytes[SIGNATURE_LENGTH]):
    # Avoid overflowing the Merkle tree (and prevent edge case in computing `self.branch`)
    assert self.deposit_count < MAX_DEPOSIT_COUNT

    # Validate deposit data
    deposit_amount: uint256 = msg.value / as_wei_value(1, "gwei")
    assert deposit_amount >= self.MIN_DEPOSIT_AMOUNT
    assert len(pubkey) == PUBKEY_LENGTH
    assert len(withdrawal_credentials) == WITHDRAWAL_CREDENTIALS_LENGTH
    assert len(signature) == SIGNATURE_LENGTH

    # Emit `DepositEvent` log
    amount: bytes[8] = self.to_little_endian_64(deposit_amount)
    log.DepositEvent(pubkey, withdrawal_credentials, amount, signature, self.to_little_endian_64(self.deposit_count))

    # Compute `DepositData` hash tree root
    zero_bytes32: bytes32 = 0x0000000000000000000000000000000000000000000000000000000000000000
    pubkey_root: bytes32 = sha256(concat(pubkey, slice(zero_bytes32, start=0, len=64 - PUBKEY_LENGTH)))
    signature_root: bytes32 = sha256(concat(
        sha256(slice(signature, start=0, len=64)),
        sha256(concat(slice(signature, start=64, len=SIGNATURE_LENGTH - 64), zero_bytes32)),
    ))
    node: bytes32 = sha256(concat(
        sha256(concat(pubkey_root, withdrawal_credentials)),
        sha256(concat(amount, slice(zero_bytes32, start=0, len=32 - AMOUNT_LENGTH), signature_root)),
    ))

    # Add `DepositData` hash tree root to Merkle tree (update a single `branch` node)
    self.deposit_count += 1
    size: uint256 = self.deposit_count
    for height in range(DEPOSIT_CONTRACT_TREE_DEPTH):
        if bitwise_and(size, 1) == 1:  # More gas efficient than `size % 2 == 1`
            self.branch[height] = node
            break
        node = sha256(concat(self.branch[height], node))
        size /= 2


# !!! DEBUG ONLY !!!
# This method is NOT part of the final ETH2.0 deposit contract, but we use it
# to recover test funds.
@public
def drain():
    send(self.drain_address, self.balance)