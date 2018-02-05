pragma solidity ^0.4.19;

contract VMC {
  event TxToShard(address indexed to, int indexed shardId, int receiptId);
  event CollationAdded(uint indexed shardId, bytes collationHeader, bool isNewHead, uint score);
  event Deposit(address validator, int index);
  event Withdraw(int validatorIndex);

  struct Validator {
    // Amount of wei the validator holds
    uint deposit;
    // The validator's address
    address addr;
  }

  struct CollationHeader {
    bytes32 parentCollationHash;
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

  mapping (int => Validator) validators;
  mapping (int => mapping (bytes32 => CollationHeader)) collationHeaders;
  mapping (int => Receipt) receipts;

  mapping (int => bytes32) shardHead;
  int numValidators;
  int numReceipts;
  // Indexs of empty slots caused by the function `withdraw`
  mapping (int => int) emptySlotsStack;
  // The top index of the stack in empty_slots_stack
  int emptySlotsStackTop;
  // Has the validator deposited before?
  mapping (address => bool) isValidatorDeposited;

  // Constant values
  uint constant periodLength = 5;
  int constant shardCount = 100;
  // The exact deposit size which you have to deposit to become a validator
  uint constant depositSize = 100 ether;
  // Number of periods ahead of current period, which the contract
  // is able to return the collator of that period
  uint constant lookAheadPeriods = 4;

  // Log the latest period number of the shard
  mapping (int => int) periodHead;

  function VMC() public {
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

  function getValidatorsMaxIndex() internal view returns(int) {
    int activateValidatorNum = 0;
    int allValidatorSlotsNum = numValidators + emptySlotsStackTop;

    // TODO: any better way to iterate the mapping?
    for (int i = 0; i < 1024; ++i) {
        if (i >= allValidatorSlotsNum)
            break;
        if (validators[i].addr != 0x0)
            activateValidatorNum += 1;
    }
    return activateValidatorNum + emptySlotsStackTop;
  }

  function deposit() public payable returns(int) {
    require(!isValidatorDeposited[msg.sender]);
    require(msg.value == depositSize);
    // Find the empty slot index in validators set
    int index;
    if (!isStackEmpty())
      index = stackPop();
    else
      index = int(numValidators);
      
    validators[index] = Validator({
      deposit: msg.value,
      addr: msg.sender
    });
  ++numValidators;
  isValidatorDeposited[msg.sender] = true;
  
  Deposit(msg.sender, index);
  return index;
  }

  function withdraw(int _validatorIndex) public {
    require(msg.sender == validators[_validatorIndex].addr);
    // [FIXME] Should consider calling the validator's contract, might be useful
    // when the validator is a contract.
    validators[_validatorIndex].addr.transfer(validators[_validatorIndex].deposit);
    isValidatorDeposited[validators[_validatorIndex].addr] = false;
    delete validators[_validatorIndex];
    stackPush(_validatorIndex);
    --numValidators;
    Withdraw(_validatorIndex);
  }

  // Uses a block hash as a seed to pseudorandomly select a signer from the validator set.
  // [TODO] Chance of being selected should be proportional to the validator's deposit.
  // Should be able to return a value for the current period or any future period up to.
  function getEligibleProposer(int _shardId, uint _period) public view returns(address) {
    require(_period >= lookAheadPeriods);
    require((_period - lookAheadPeriods) * periodLength < block.number);
    require(numValidators > 0);
    // [TODO] Should check further if this safe or not
    return validators[
      int(
      uint(keccak256(uint(block.blockhash(_period - lookAheadPeriods)) * periodLength, _shardId))
      %
      uint(getValidatorsMaxIndex())
      )].addr;
  }

  struct HeaderVars {
    bytes32 entireHeaderHash;
    int score;
    address validatorAddr;
    bool isNewHead;
  }
  function addHeader(int _shardId, uint _expectedPeriodNumber, bytes32 _periodStartPrevHash,
                     bytes32 _parentCollationHash, bytes32 _txListRoot, address _collationCoinbase,
                     bytes32 _postStateRoot, bytes32 _receiptRoot, int _collationNumber) public returns(bool) {
    HeaderVars memory headerVars;

    // Check if the header is valid
    require((_shardId >= 0) && (_shardId < shardCount));
    require(block.number >= periodLength);
    require(_expectedPeriodNumber == block.number / periodLength);
    require(_periodStartPrevHash == block.blockhash(_expectedPeriodNumber * periodLength - 1));

    // Check if this header already exists
    headerVars.entireHeaderHash = keccak256(_shardId, _expectedPeriodNumber, _periodStartPrevHash,
                                   _parentCollationHash, _txListRoot, bytes32(_collationCoinbase),
                                   _postStateRoot, _receiptRoot, _collationNumber);
    assert(headerVars.entireHeaderHash != 0x0);
    assert(collationHeaders[_shardId][headerVars.entireHeaderHash].score == 0);
    // Check whether the parent exists.
    // if (parent_collation_hash == 0), i.e., is the genesis,
    // then there is no need to check.
    if (_parentCollationHash != 0x0)
        assert((_parentCollationHash == 0x0) || (collationHeaders[_shardId][_parentCollationHash].score > 0));
    // Check if only one collation in one period
    assert(periodHead[_shardId] < int(_expectedPeriodNumber));

    // Check the signature with validation_code_addr
    headerVars.validatorAddr = getEligibleProposer(_shardId, block.number/periodLength);
    require(headerVars.validatorAddr != 0x0);
    require(msg.sender == headerVars.validatorAddr);

    // Check score == collationNumber
    headerVars.score = collationHeaders[_shardId][_parentCollationHash].score + 1;
    require(_collationNumber == headerVars.score);

    // Add the header
    collationHeaders[_shardId][headerVars.entireHeaderHash] = CollationHeader({
      parentCollationHash: _parentCollationHash,
      score: headerVars.score
    });

    // Update the latest period number
    periodHead[_shardId] = int(_expectedPeriodNumber);

    // Determine the head
    if (headerVars.score > collationHeaders[_shardId][shardHead[_shardId]].score) {
      shardHead[_shardId] = headerVars.entireHeaderHash;
      headerVars.isNewHead = true;
    }
    // [TODO] Log
    //CollationAdded(headerBytes, isNewHead, _score);

    return true;
  }

  // Returns the gas limit that collations can currently have (by default make
  // this function always answer 10 million).
  function getCollationGasLimit() public pure returns(uint) {
    return 10000000;
  }

  // Records a request to deposit msg.value ETH to address to in shard shard_id
  // during a future collation. Saves a `receipt ID` for this request,
  // also saving `msg.sender`, `msg.value`, `to`, `shard_id`, `startgas`,
  // `gasprice`, and `data`.
  function txToShard(address _to, int _shardId, uint _txStartgas, uint _txGasprice, bytes12 _data) public payable returns(int) {
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
  
  function updataGasPrice(int _receiptId, uint _txGasprice) public payable returns(bool) {
    require(receipts[_receiptId].sender == msg.sender);
    receipts[_receiptId].txGasprice = _txGasprice;
    return true;
  }
}
