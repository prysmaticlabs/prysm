pragma solidity ^0.4.19;

contract VMC {
  event TxToShard(address indexed to, int indexed shardId, int receiptId);
  event CollationAdded(uint256 indexed shardId, bytes collationHeader, bool isNewHead, uint256 score);
  event Deposit(address validator, int index);
  event Withdraw(int validatorIndex);

  struct Validator {
    // Amount of wei the validator holds
    uint deposit;
    // The validator's address
    address addr;
    // The cycle number which the validator would be included after
    int cycle;
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
    address sender;
    address to;
    bytes32 data;
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
  int constant lookAheadPeriods = 4;

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
        if (validators[i].cycle != 0x0)
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
  function getEligibleProposer(int shardId, int period) public {
    require(period >= lookAheadPeriods);
    require((period - lookAheadPeriods) * periodLength < block.number);
    require(numValidators > 0);
    return validators[
      uint(keccak256(block.blockhash(period - lookAheadPeriods) * periodLength, shardId))
      %
      uint(getValidatorsMaxIndex())
      ].addr;
  }

  struct Header {
      int shardId;
      uint expectedPeriodNumber;
      bytes32 periodStartPrevhash;
      bytes32 parentCollationHash;
      bytes32 txListRoot;
      address collationCoinbase;
      bytes32 postStateRoot;
      bytes32 receiptRoot;
      int collationNumber;
      bytes sig;
    }

  function addHeader(int shardId, int expectedPeriodNumber, bytes32 periodStartPrevHash,
                     bytes32 parentCollationHash, bytes32 txListRoot, address collationCoinbase,
                     bytes32 postStateRoot, bytes32 receiptRoot, int collationNumber) public returns(bool) {

    // Check if the header is valid
    require((shardId >= 0) && (shardId < shardCount));
    require(block.number >= periodLength);
    require(expectedPeriodNumber == (block.number / periodLength));
    require(periodStartPrevHash == block.blockhash(expectedPeriodNumber * periodLength - 1));

    // Check if this header already exists
    var entireHeaderHash = keccak256(shardId, expectedPeriodNumber, periodStartPrevHash,
                                   parentCollationHash, txListRoot, bytes32(collationCoinbase),
                                   postStateRoot, receiptRoot, collationNumber);
    assert(entireHeaderHash != 0x0);
    assert(collationHeaders[shardId][entireHeaderHash].score == 0);
    // Check whether the parent exists.
    // if (parent_collation_hash == 0), i.e., is the genesis,
    // then there is no need to check.
    if (parentCollationHash != 0x0)
        assert((parentCollationHash == 0x0) || (collationHeaders[shardId][parentCollationHash].score > 0));
    // Check if only one collation in one period
    assert(periodHead[shardId] < int(expectedPeriodNumber));

    // Check the signature with validation_code_addr
    var validatorAddr = getEligibleProposer(shardId, block.number/periodLength);
    require(validatorAddr != 0x0);
    require(msg.sender == validatorAddr);

    // Check score == collationNumber
    var _score = collationHeaders[shardId][parentCollationHash].score + 1;
    require(collationNumber == _score);

    // Add the header
    collationHeaders[shardId][entireHeaderHash] = CollationHeader({
      parentCollationHash: parentCollationHash,
      score: _score
    });

    // Update the latest period number
    periodHead[shardId] = int(expectedPeriodNumber);

    // Determine the head
    var isNewHead = false;
    if (_score > collationHeaders[shardId[shardHead[shardId]]].score) {
      shardHead[shardId] = entireHeaderHash;
      isNewHead = true;
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
