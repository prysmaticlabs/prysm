pragma solidity ^0.4.19;

contract SMC {
  event TxToShard(address indexed to, int indexed shardId, int receiptId);
  event CollationAdded(int indexed shardId, uint expectedPeriodNumber, 
                     bytes32 periodStartPrevHash, bytes32 parentHash,
                     bytes32 transactionRoot, address coinbase,
                     bytes32 stateRoot, bytes32 receiptRoot,
                     int number, bool isNewHead, int score);
  event Deposit(address collator, int index);
  event Withdraw(int index);

  struct Collator {
    // Amount of wei the collator holds
    uint deposit;
    // The collator's address
    address addr;
  }

  struct CollationHeader {
    bytes32 parentHash;
    int score;
  }

  struct Receipt {
    int shardId;
    uint txStartgas;
    uint txGasprice;
    uint value;
    bytes32 data;
    address sender;
    address to;
  }

  // Packed variables to be used in addHeader
  struct HeaderVars {
    bytes32 entireHeaderHash;
    int score;
    address collatorAddr;
    bool isNewHead;
  }

  // collatorId => Collators
  mapping (int => Collator) public collators;
  // shardId => (headerHash => CollationHeader)
  mapping (int => mapping (bytes32 => CollationHeader)) public collationHeaders;
  // receiptId => Receipt
  mapping (int => Receipt) public receipts;
  // shardId => headerHash
  mapping (int => bytes32) shardHead;

  // Number of collators
  int public numCollators;
  // Number of receipts
  int numReceipts;
  // Indexs of empty slots caused by the function `withdraw`
  mapping (int => int) emptySlotsStack;
  // The top index of the stack in empty_slots_stack
  int emptySlotsStackTop;
  // Has the collator deposited before?
  mapping (address => bool) public isCollatorDeposited;

  // Constant values
  uint constant periodLength = 5;
  int constant public shardCount = 100;
  // The exact deposit size which you have to deposit to become a collator
  uint constant depositSize = 1000 ether;
  // Number of periods ahead of current period, which the contract
  // is able to return the collator of that period
  uint constant lookAheadPeriods = 4;

  // Log the latest period number of the shard
  mapping (int => int) public periodHead;

  function SMC() public {
  }

  // Returns the gas limit that collations can currently have (by default make
  // this function always answer 10 million).
  function getCollationGasLimit() public pure returns(uint) {
    return 10000000;
  }

  // Uses a block hash as a seed to pseudorandomly select a signer from the collator pool.
  // [TODO] Chance of being selected should be proportional to the collator's deposit.
  // Should be able to return a value for the current period or any future period up to.
  function getEligibleCollator(int _shardId, uint _period) public view returns(address) {
    require(_period >= lookAheadPeriods);
    require((_period - lookAheadPeriods) * periodLength < block.number);
    require(numCollators > 0);
    // [TODO] Should check further if this safe or not
    return collators[
      int(
        uint(
          keccak256(
            uint(block.blockhash((_period - lookAheadPeriods) * periodLength)),
            _shardId
          )
        ) %
        uint(getCollatorsMaxIndex())
      )
    ].addr;
  }

  function deposit() public payable returns(int) {
    require(!isCollatorDeposited[msg.sender]);
    require(msg.value == depositSize);
    // Find the empty slot index in collators pool
    int index;
    if (!isStackEmpty())
      index = stackPop();
    else
      index = int(numCollators);

    collators[index] = Collator({
      deposit: msg.value,
      addr: msg.sender
    });
    ++numCollators;
    isCollatorDeposited[msg.sender] = true;

    Deposit(msg.sender, index);
    return index;
  }

  // Removes the collator from the collator pool and refunds the deposited ether
  function withdraw(int _collatorIndex) public {
    require(msg.sender == collators[_collatorIndex].addr);
    // [FIXME] Should consider calling the collator's contract, might be useful
    // when the collator is a contract.
    collators[_collatorIndex].addr.transfer(collators[_collatorIndex].deposit);
    isCollatorDeposited[collators[_collatorIndex].addr] = false;
    delete collators[_collatorIndex];
    stackPush(_collatorIndex);
    --numCollators;
    Withdraw(_collatorIndex);
  }

  // Attempts to process a collation header, returns true on success, reverts on failure.
  function addHeader(int _shardId, uint _expectedPeriodNumber, bytes32 _periodStartPrevHash,
                     bytes32 _parentHash, bytes32 _transactionRoot,
                     address _coinbase, bytes32 _stateRoot, bytes32 _receiptRoot,
                     int _number) public returns(bool) {
    HeaderVars memory headerVars;

    // Check if the header is valid
    require((_shardId >= 0) && (_shardId < shardCount));
    require(block.number >= periodLength);
    require(_expectedPeriodNumber == block.number / periodLength);
    require(_periodStartPrevHash == block.blockhash(_expectedPeriodNumber * periodLength - 1));

    // Check if this header already exists
    headerVars.entireHeaderHash = keccak256(_shardId, _expectedPeriodNumber, _periodStartPrevHash,
                                   _parentHash, _transactionRoot, bytes32(_coinbase),
                                   _stateRoot, _receiptRoot, _number);
    assert(collationHeaders[_shardId][headerVars.entireHeaderHash].score == 0);
    // Check whether the parent exists.
    // if (parent_collation_hash == 0), i.e., is the genesis,
    // then there is no need to check.
    if (_parentHash != 0x0)
        assert(collationHeaders[_shardId][_parentHash].score > 0);
    // Check if only one collation in one period
    assert(periodHead[_shardId] < int(_expectedPeriodNumber));

    // Check the signature with validation_code_addr
    headerVars.collatorAddr = getEligibleCollator(_shardId, block.number/periodLength);
    require(headerVars.collatorAddr != 0x0);
    require(msg.sender == headerVars.collatorAddr);

    // Check score == collationNumber
    headerVars.score = collationHeaders[_shardId][_parentHash].score + 1;
    require(_number == headerVars.score);

    // Add the header
    collationHeaders[_shardId][headerVars.entireHeaderHash] = CollationHeader({
      parentHash: _parentHash,
      score: headerVars.score
    });

    // Update the latest period number
    periodHead[_shardId] = int(_expectedPeriodNumber);

    // Determine the head
    if (headerVars.score > collationHeaders[_shardId][shardHead[_shardId]].score) {
      shardHead[_shardId] = headerVars.entireHeaderHash;
      headerVars.isNewHead = true;
    }

    CollationAdded(_shardId, _expectedPeriodNumber, _periodStartPrevHash,
                   _parentHash, _transactionRoot, _coinbase, _stateRoot, 
                   _receiptRoot, _number, headerVars.isNewHead, headerVars.score);

    return true;
  }

  // Records a request to deposit msg.value ETH to address to in shard shard_id
  // during a future collation. Saves a `receipt ID` for this request,
  // also saving `msg.sender`, `msg.value`, `to`, `shard_id`, `startgas`,
  // `gasprice`, and `data`.
  function txToShard(address _to, int _shardId, uint _txStartgas, uint _txGasprice,
                     bytes12 _data) public payable returns(int) {
    receipts[numReceipts] = Receipt({
      shardId: _shardId,
      txStartgas: _txStartgas,
      txGasprice: _txGasprice,
      value: msg.value,
      sender: msg.sender,
      to: _to,
      data: _data
    });
    var receiptId = numReceipts;
    ++numReceipts;

    TxToShard(_to, _shardId, receiptId);
    return receiptId;
  }

  function updateGasPrice(int _receiptId, uint _txGasprice) public payable returns(bool) {
    require(receipts[_receiptId].sender == msg.sender);
    receipts[_receiptId].txGasprice = _txGasprice;
    return true;
  }

  function isStackEmpty() internal view returns(bool) {
    return emptySlotsStackTop == 0;
  }

  function stackPush(int index) internal {
    emptySlotsStack[emptySlotsStackTop] = index;
    ++emptySlotsStackTop;
  }

  function stackPop() internal returns(int) {
    if (isStackEmpty())
      return -1;
    --emptySlotsStackTop;
    return emptySlotsStack[emptySlotsStackTop];
  }

  function getCollatorsMaxIndex() internal view returns(int) {
    int activateCollatorNum = 0;
    int allCollatorSlotsNum = numCollators + emptySlotsStackTop;

    // TODO: any better way to iterate the mapping?
    for (int i = 0; i < 1024; ++i) {
        if (i >= allCollatorSlotsNum)
            break;
        if (collators[i].addr != 0x0)
            activateCollatorNum += 1;
    }
    return activateCollatorNum + emptySlotsStackTop;
  }
}
