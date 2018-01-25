pragma solidity ^0.4.19;

import "RLP.sol";

interface ValidatorContract {
  function withdraw(bytes32 _sig) returns(bool);
}

interface SigHasherContract {
  // function () returns(bytes32);
}

contract VMC {
  using RLP for RLP.RLPItem;
  using RLP for RLP.Iterator;
  using RLP for bytes;
  
  struct Validator {
    // Amount of wei the validator holds
    uint deposit;
    // The address which the validator's signatures must verify to (to be later replaced with validation code)
    ValidatorContract validationCodeAddr;
    // Addess to withdraw to
    address returnAddr;
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

  // The exact deposit size which you have to deposit to become a validator
  uint depositSize;
  // Any given validator randomly gets allocated to some number of shards every SHUFFLING_CYCLE
  int shufflingCycleLength;
  // Gas limit of the signature validation code
  uint sigGasLimit;
  // Is a valcode addr deposited now?
  mapping (address => bool) isValcodeDeposited;
  uint periodLength;
  int numValidatorsPerCycle;
  int shardCount;
  bytes32 addHeaderLogTopic;
  SigHasherContract sighasher;
  // Log the latest period number of the shard
  mapping (int => int) periodHead;

  function VMC() public {
    numValidators = 0;
    emptySlotsStackTop = 0;
    depositSize = 100 ether;
    shufflingCycleLength = 5; // FIXME: just modified temporarily for test;
    sigGasLimit = 400000;
    periodLength = 5;
    numValidatorsPerCycle = 100;
    shardCount = 100;
    addHeaderLogTopic = keccak256("add_header()");
    sighasher = SigHasherContract(0xDFFD41E18F04Ad8810c83B14FD1426a82E625A7D);
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

  function getValidatorsMaxIndex() public view returns(int) {
    address zeroAddr = 0x0;
    int activateValidatorNum = 0;
    int currentCycle = int(block.number) / shufflingCycleLength;
    int allValidatorSlotsNum = numValidators + emptySlotsStackTop;

    // TODO: any better way to iterate the mapping?
    for (int i = 0; i < 1024; ++i) {
        if (i >= allValidatorSlotsNum)
            break;
        if ((validators[i].validationCodeAddr != zeroAddr) &&
            (validators[i].cycle <= currentCycle))
            activateValidatorNum += 1;
    }
    return activateValidatorNum + emptySlotsStackTop;
  }

  function deposit(ValidatorContract _validationCodeAddr, address _returnAddr) public payable returns(int) {
    require(!isValcodeDeposited[_validationCodeAddr]);
    require(msg.value == depositSize);
    // Find the empty slot index in validators set
    int index;
    int nextCycle = 0;
    if (!isStackEmpty())
      index = stackPop();
    else {
      index = int(numValidators);
      nextCycle = (int(block.number) / shufflingCycleLength) + 1;
      validators[index] = Validator({
        deposit: msg.value,
        validationCodeAddr: _validationCodeAddr,
        returnAddr: _returnAddr,
        cycle: nextCycle
      });
    }
    ++numValidators;
    isValcodeDeposited[_validationCodeAddr] = true;
    
    log2(keccak256("deposit()"), bytes32(address(_validationCodeAddr)), bytes32(index));
    return index;
  }

  function withdraw(int _validatorIndex, bytes32 _sig) public returns(bool) {
    var msgHash = keccak256("withdraw");
    // TODO: Check this function
    var result = validators[_validatorIndex].validationCodeAddr.withdraw.gas(sigGasLimit)(_sig);
    if (result) {
      validators[_validatorIndex].returnAddr.transfer(validators[_validatorIndex].deposit);
      isValcodeDeposited[validators[_validatorIndex].validationCodeAddr] = false;
      delete validators[_validatorIndex];
      stackPush(_validatorIndex);
      --numValidators;
      log1(msgHash, bytes32(_validatorIndex));
      return result;
    }
  }

  function sample(int _shardId) public constant returns(address) {
    require(block.number >= periodLength);
    var cycle = int(block.number) / shufflingCycleLength;
    int cycleStartBlockNumber = cycle * shufflingCycleLength - 1;
    if (cycleStartBlockNumber < 0)
      cycleStartBlockNumber = 0;
    int cycleSeed = int(block.blockhash(uint(cycleStartBlockNumber)));
    // originally, error occurs when block.number <= 4 because
    // `seed_block_number` becomes negative in these cases.
    int seed = int(block.blockhash(block.number - (block.number % uint(periodLength)) - 1));

    uint indexInSubset = uint(keccak256(seed, bytes32(_shardId))) % uint(numValidatorsPerCycle);
    uint validatorIndex = uint(keccak256(cycleSeed, bytes32(_shardId), bytes32(indexInSubset))) % uint(getValidatorsMaxIndex());
    
    if (validators[int(validatorIndex)].cycle > cycle)
      return 0x0;
    else
      return validators[int(validatorIndex)].validationCodeAddr;
  }

  // Get all possible shard ids that the given _valcodeAddr
  // may be sampled in the current cycle
  function getShardList(ValidatorContract _valcodeAddr) public constant returns(bool[100]) {
    bool[100] memory shardList;
    int cycle = int(block.number) / shufflingCycleLength;
    int cycleStartBlockNumber = cycle * shufflingCycleLength - 1;
    if (cycleStartBlockNumber < 0)
      cycleStartBlockNumber = 0;

    var cycleSeed = block.blockhash(uint(cycleStartBlockNumber));
    int validatorsMaxIndex = getValidatorsMaxIndex();
    if (numValidators != 0) {
      for (uint8 shardId = 0; shardId < 100; ++shardId) {
        shardList[shardId] = false;
        for (uint8 possibleIndexInSubset = 0; possibleIndexInSubset < 100; ++possibleIndexInSubset) {
          uint validatorIndex = uint(keccak256(cycleSeed, bytes32(shardId), bytes32(possibleIndexInSubset))) 
                             % uint(validatorsMaxIndex);
          if (_valcodeAddr == validators[int(validatorIndex)].validationCodeAddr) {
            shardList[shardId] = true;
            break;
          }
        }
      }
    }
    return shardList;
  }

  // function checkHeader(int _shardId, bytes32 _periodStartPrevhash, int _expectedPeriodNumber) internal {
  //   // Check if the header is valid
  //   assert(_shardId >= 0 && _shardId < shardCount);
  //   assert(block.number >= periodLength);
  //   assert(uint(_expectedPeriodNumber) == block.number / periodLength);
  //   assert(_periodStartPrevhash == block.blockhash(uint(_expectedPeriodNumber)*periodLength - 1));

  //   // Check if this header already exists
  //   var entireHeaderHash = keccak256(_header);
  //   assert(entireHeaderHash != bytes32(0));
  //   assert(collationHeaders[shardId][entireHeaderHash].score == 0);
  // }

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

  function addHeader(bytes _header) public returns(bool) {
    address zeroAddr = 0x0;
    // require(_header.length <= 4096);
    // TODO
    // values = RLPList(header, [num, num, bytes32, bytes32, bytes32, address, bytes32, bytes32, num, bytes])
    // return True
    bytes memory mHeader = _header;
    var RLPList = mHeader.toRLPItem(true).iterator();
    var header = Header({
      shardId: RLPList.next().toInt(),
      expectedPeriodNumber: RLPList.next().toUint(),
      periodStartPrevhash: RLPList.next().toBytes32(),
      parentCollationHash: RLPList.next().toBytes32(),
      txListRoot: RLPList.next().toBytes32(),
      collationCoinbase: RLPList.next().toAddress(),
      postStateRoot: RLPList.next().toBytes32(),
      receiptRoot: RLPList.next().toBytes32(),
      collationNumber: RLPList.next().toInt(),
      sig: RLPList.next().toBytes()
    });

    // Check if the header is valid
    require((header.shardId >= 0) && (header.shardId < shardCount));
    require(block.number >= periodLength);
    require(header.expectedPeriodNumber == (block.number / periodLength));
    require(header.periodStartPrevhash == block.blockhash(header.expectedPeriodNumber * periodLength - 1));


    // Check if this header already exists
    var entireHeaderHash = keccak256(header);
    assert(entireHeaderHash != 0x0);
    assert(collationHeaders[header.shardId][entireHeaderHash].score == 0);
    // Check whether the parent exists.
    // if (parent_collation_hash == 0), i.e., is the genesis,
    // then there is no need to check.
    if (header.parentCollationHash != 0x0)
        assert ((header.parentCollationHash == 0x0) || (collationHeaders[header.shardId][header.parentCollationHash].score > 0));
    // Check if only one colllation in one period
    assert (periodHead[header.shardId] < int(header.expectedPeriodNumber));

    // Check the signature with validation_code_addr
    var collatorValcodeAddr = sample(header.shardId);
    if (collatorValcodeAddr == 0x0)
        return false;
    
    // assembly {
      // TODO next block
    // }
    // sighash = extract32(raw_call(self.sighasher_addr, header, gas=200000, outsize=32), 0)
    // assert extract32(raw_call(collator_valcode_addr, concat(sighash, sig), gas=self.sig_gas_limit, outsize=32), 0) == as_bytes32(1)

    // Check score == collation_number
    var _score = collationHeaders[header.shardId][header.parentCollationHash].score + 1;
    assert(header.collationNumber == _score);


    // Add the header
    collationHeaders[header.shardId][entireHeaderHash] = CollationHeader({
      parentCollationHash: header.parentCollationHash,
      score: _score
    });

    // Update the latest period number
    periodHead[header.shardId] = int(header.expectedPeriodNumber);

    // Determine the head
    if (_score > collationHeaders[header.shardId][shardHead[header.shardId]].score) {
      var previousHeadHash = shardHead[header.shardId];
      shardHead[header.shardId] = entireHeaderHash;
      // Logs only when `change_head` happens due to the fork
      // TODO LOG
      // log1([addHeaderLogTopic, keccak256("change_head"), entireHeaderHash], previousHeadHash);
    }
    // Emit log
    // TODO LOG
    // log1(addHeaderLogTopic, _header);
    
    return true;
  }

  function getPeriodStartPrevhash(uint _expectedPeriodNumber) public constant returns(bytes32) {
    uint blockNumber = _expectedPeriodNumber * periodLength - 1;
    require(block.number > blockNumber);
    return block.blockhash(blockNumber);
  }



  // Returns the difference between the block number of this hash and the block
  // number of the 10000th ancestor of this hash.
  function getAncestorDistance(bytes32 /*_hash*/) public pure returns(bytes32) {
    // TODO
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
    
    log3(keccak256("tx_to_shard()"), bytes32(_to), bytes32(_shardId), bytes32(receiptId));
    return receiptId;
  }
  
  function updataGasPrice(int _receiptId, uint _txGasprice) public payable returns(bool) {
    require(receipts[_receiptId].sender == msg.sender);
    receipts[_receiptId].txGasprice = _txGasprice;
    return true;
  }
}
