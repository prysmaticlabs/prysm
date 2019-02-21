## compiled with v0.1.0-beta.8 ##
DEPOSIT_TREE_DEPTH: constant(uint256) = 32
TWO_TO_POWER_OF_TREE_DEPTH: constant(uint256) = 4294967296  # 2**32
SECONDS_PER_DAY: constant(uint256) = 86400
MAX_64_BIT_VALUE: constant(uint256) = 18446744073709551615  # 2**64 - 1

Deposit: event({deposit_root: bytes32, data: bytes[528], merkle_tree_index: bytes[8], branch: bytes32[32]})
ChainStart: event({deposit_root: bytes32, time: bytes[8]})

CHAIN_START_FULL_DEPOSIT_THRESHOLD: public(uint256)
MIN_DEPOSIT_AMOUNT: public(uint256) # Gwei
MAX_DEPOSIT_AMOUNT: public(uint256) # Gwei
zerohashes: bytes32[32]
branch: bytes32[32]
deposit_count: public(uint256)
full_deposit_count: public(uint256)
custom_chainstart_delay: public(uint256)
genesisTime: public(bytes[8])
drain_address: public(address)

@public
def __init__( # Parameters for debugging, not for production use!
        depositThreshold: uint256, 
        minDeposit: uint256,
        maxDeposit: uint256, 
        customChainstartDelay: uint256,
        _drain_address: address):
    self.CHAIN_START_FULL_DEPOSIT_THRESHOLD = depositThreshold
    self.MIN_DEPOSIT_AMOUNT = minDeposit
    self.MAX_DEPOSIT_AMOUNT = maxDeposit
    self.custom_chainstart_delay = customChainstartDelay
    self.drain_address = _drain_address
    for i in range(31):
        self.zerohashes[i+1] = sha3(concat(self.zerohashes[i], self.zerohashes[i]))
        self.branch[i+1] = self.zerohashes[i+1]

@private
@constant
def to_bytes8(value: uint256) -> bytes[8]:
    return slice(convert(value, bytes32), start=24, len=8)

@public
@constant
def to_little_endian_64(value: uint256) -> bytes[8]:
    assert value <= MAX_64_BIT_VALUE

    big_endian_64: bytes[8] = self.to_bytes8(value)

    # array access for bytes[] not currently supported in vyper so
    # reversing bytes using bitwise uint256 manipulations
    x: uint256 = convert(big_endian_64, uint256)
    y: uint256 = 0
    for i in range(8):
        y = shift(y, 8)
        y = y + bitwise_and(x, 255)
        x = shift(x, -8)

    return self.to_bytes8(y)

@public
@constant
def get_deposit_root() -> bytes32:
    root:bytes32 = 0x0000000000000000000000000000000000000000000000000000000000000000
    size:uint256 = self.deposit_count
    for h in range(32):
        if size % 2 == 1:
            root = sha3(concat(self.branch[h], root))
        else:
            root = sha3(concat(root, self.zerohashes[h]))
        size /= 2
    return root

@payable
@public
def deposit(deposit_input: bytes[512]):
    deposit_amount: uint256 = msg.value / as_wei_value(1, "gwei")

    assert deposit_amount >= self.MIN_DEPOSIT_AMOUNT
    assert deposit_amount <= self.MAX_DEPOSIT_AMOUNT

    index: uint256 = self.deposit_count
    deposit_timestamp: uint256 = as_unitless_number(block.timestamp)
    deposit_data: bytes[528] = concat(self.to_little_endian_64(deposit_amount), self.to_little_endian_64(deposit_timestamp), deposit_input)

    # add deposit to merkle tree
    i: int128 = 0
    power_of_two: uint256 = 2
    for _ in range(32):
        if (index+1) % power_of_two != 0:
            break
        i += 1
        power_of_two *= 2
    value:bytes32 = sha3(deposit_data)
    for j in range(32):
        if j < i:
            value = sha3(concat(self.branch[j], value))
    self.branch[i] = value

    self.deposit_count += 1
    new_deposit_root: bytes32 = self.get_deposit_root()
    log.Deposit(new_deposit_root, deposit_data, self.to_little_endian_64(index), self.branch)

    if deposit_amount == self.MAX_DEPOSIT_AMOUNT:
        self.full_deposit_count += 1
        if self.full_deposit_count == self.CHAIN_START_FULL_DEPOSIT_THRESHOLD:
            if self.custom_chainstart_delay > 0:
                timestamp_boundary: uint256 = as_unitless_number(block.timestamp) - as_unitless_number(block.timestamp) % self.custom_chainstart_delay + self.custom_chainstart_delay
                self.genesisTime = self.to_little_endian_64(timestamp_boundary)
                log.ChainStart(self.get_deposit_root(), self.genesisTime)
            else:
                timestamp_day_boundary: uint256 = as_unitless_number(block.timestamp) - as_unitless_number(block.timestamp) % SECONDS_PER_DAY + SECONDS_PER_DAY
                self.genesisTime = self.to_little_endian_64(timestamp_day_boundary)
                log.ChainStart(new_deposit_root, self.genesisTime)


# !!! DEBUG ONLY !!!
# This method is NOT part of the final ETH2.0 deposit contract, but we use it 
# to recover test funds.
@public
def drain():
    send(self.drain_address, self.balance)
