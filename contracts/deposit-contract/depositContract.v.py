## compiled with v0.1.0-beta.7 ##

MIN_DEPOSIT: constant(uint256(wei)) = as_wei_value(1000000000 , "gwei")
MAX_DEPOSIT: constant(uint256(wei)) = as_wei_value(32000000000, "gwei")
GWEI_PER_ETH: constant(uint256) = 1000000000  # 10**9
DEPOSIT_CONTRACT_TREE_DEPTH: constant(uint256) = 32
TWO_TO_POWER_OF_TREE_DEPTH: constant(uint256) = 4294967296  # 2**32
SECONDS_PER_DAY: constant(uint256) = 86400

Deposit: event({previous_deposit_root: bytes32, data: bytes[2064], merkle_tree_index: bytes[8]})
ChainStart: event({deposit_root: bytes32, time: bytes[8]})

CHAIN_START_FULL_DEPOSIT_THRESHOLD: uint256
deposit_tree: map(uint256, bytes32)
deposit_count: uint256
full_deposit_count: uint256

@public
def __init__(depositThreshold: uint256):
    self.CHAIN_START_FULL_DEPOSIT_THRESHOLD = depositThreshold

@payable
@public
def deposit(deposit_input: bytes[2048]):
    assert msg.value >= MIN_DEPOSIT
    assert msg.value <= MAX_DEPOSIT

    index: uint256 = self.deposit_count + TWO_TO_POWER_OF_TREE_DEPTH
    deposit_amount: bytes[8] = slice(concat("", convert(msg.value / GWEI_PER_ETH, bytes32)), start=24, len=8)
    deposit_timestamp: bytes[8] = slice(concat("", convert(block.timestamp, bytes32)), start=24, len=8)
    deposit_data: bytes[2064] = concat(deposit_amount, deposit_timestamp, deposit_input)
    merkle_tree_index: bytes[8] = slice(concat("", convert(index, bytes32)), start=24, len=8)

    log.Deposit(self.deposit_tree[1], deposit_data, merkle_tree_index)

    # add deposit to merkle tree
    self.deposit_tree[index] = sha3(deposit_data)
    for i in range(DEPOSIT_CONTRACT_TREE_DEPTH):
        index /= 2
        self.deposit_tree[index] = sha3(concat(self.deposit_tree[index * 2], self.deposit_tree[index * 2 + 1]))

    self.deposit_count += 1
    if msg.value == MAX_DEPOSIT:
        self.full_deposit_count += 1
        if self.full_deposit_count == self.CHAIN_START_FULL_DEPOSIT_THRESHOLD:
            timestamp_day_boundary: uint256 = as_unitless_number(block.timestamp) - as_unitless_number(block.timestamp) % SECONDS_PER_DAY + SECONDS_PER_DAY
            chainstart_time: bytes[8] = slice(concat("", convert(timestamp_day_boundary, bytes32)), start=24, len=8)
            log.ChainStart(self.deposit_tree[1], chainstart_time)

@public
@constant
def get_deposit_root() -> bytes32:
    return self.deposit_tree[1]

@public
@constant
def get_branch(leaf: uint256) -> bytes32[32]: # size is DEPOSIT_CONTRACT_TREE_DEPTH (symbolic const not supported)
    branch: bytes32[32] # size is DEPOSIT_CONTRACT_TREE_DEPTH
    index: uint256 = leaf + TWO_TO_POWER_OF_TREE_DEPTH
    for i in range(DEPOSIT_CONTRACT_TREE_DEPTH):
        branch[i] = self.deposit_tree[bitwise_xor(index, 1)]
        index /= 2
    return branch