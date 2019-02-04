## compiled with v0.1.0-beta.7 ##

DEPOSIT_CONTRACT_TREE_DEPTH: constant(uint256) = 32
TWO_TO_POWER_OF_TREE_DEPTH: constant(uint256) = 4294967296  # 2**32
SECONDS_PER_DAY: constant(uint256) = 86400

Deposit: event({previous_deposit_root: bytes32, data: bytes[2064], merkle_tree_index: bytes[8]})
ChainStart: event({deposit_root: bytes32, time: bytes[8]})

MIN_DEPOSIT_AMOUNT: uint256
MAX_DEPOSIT_AMOUNT: uint256
CHAIN_START_FULL_DEPOSIT_THRESHOLD: uint256
deposit_tree: map(uint256, bytes32)
deposit_count: uint256
full_deposit_count: uint256

@public
def __init__(depositThreshold: uint256,minDeposit: uint256,maxDeposit: uint256):
    self.CHAIN_START_FULL_DEPOSIT_THRESHOLD = depositThreshold
    self.MIN_DEPOSIT_AMOUNT = minDeposit
    self.MAX_DEPOSIT_AMOUNT = maxDeposit


@private
@constant
def to_bytes(value: uint256) -> bytes[8]:
    return slice(concat("", convert(value, bytes32)), start=24, len=8)

@public
@constant
def get_deposit_root() -> bytes32:
    return self.deposit_tree[1]

@payable
@public
def deposit(deposit_input: bytes[2048]):
    deposit_amount: uint256 = msg.value / as_wei_value(1, "gwei")

    assert deposit_amount >= self.MIN_DEPOSIT_AMOUNT
    assert deposit_amount <= self.MAX_DEPOSIT_AMOUNT

    deposit_timestamp: uint256 = as_unitless_number(block.timestamp)
    deposit_data: bytes[2064] = concat(self.to_bytes(deposit_amount), self.to_bytes(deposit_timestamp), deposit_input)
    index: uint256 = self.deposit_count + TWO_TO_POWER_OF_TREE_DEPTH

    log.Deposit(self.get_deposit_root(), deposit_data, self.to_bytes(index))

    # Add deposit to merkle tree
    self.deposit_tree[index] = sha3(deposit_data)
    for i in range(DEPOSIT_CONTRACT_TREE_DEPTH):
        index /= 2
        self.deposit_tree[index] = sha3(concat(self.deposit_tree[index * 2], self.deposit_tree[index * 2 + 1]))

    self.deposit_count += 1
    if deposit_amount == self.MAX_DEPOSIT_AMOUNT:
        self.full_deposit_count += 1
        if self.full_deposit_count == self.CHAIN_START_FULL_DEPOSIT_THRESHOLD:
            # Temporarily commenting out the day boundary. 
            #timestamp_day_boundary: uint256 = deposit_timestamp - deposit_timestamp % SECONDS_PER_DAY + SECONDS_PER_DAY
            #log.ChainStart(self.get_deposit_root(), self.to_bytes(timestamp_day_boundary))
            log.ChainStart(self.get_deposit_root(), self.to_bytes(deposit_timestamp))

@public
@constant
def get_branch(leaf: uint256) -> bytes32[DEPOSIT_CONTRACT_TREE_DEPTH]:
    branch: bytes32[32] # size is DEPOSIT_CONTRACT_TREE_DEPTH
    index: uint256 = leaf + TWO_TO_POWER_OF_TREE_DEPTH
    for i in range(DEPOSIT_CONTRACT_TREE_DEPTH):
        branch[i] = self.deposit_tree[bitwise_xor(index, 1)]
        index /= 2
    return branch
