pragma solidity ^0.4.19;

contract SMC {
  event HeaderAdded(uint indexed shard_id, bytes32 chunk_root,
                      int128 period, address proposer_address);
  event NotaryRegistered(address notary, uint pool_index);
  event NotaryDeregistered(address notary, uint pool_index, uint deregistered_period);
  event NotaryReleased(address notary, uint pool_index);

  struct Notary {
    uint deregisteredPeriod;
    uint poolIndex;
    bool deposited;
  }

  struct CollationHeader {
    uint shardId;             // Number of the shard ID
    bytes32 chunkRoot;        // Root hash of the collation body
    uint period;              // Period which header should be included
    address proposerAddress;
  }

  // Packed variables to be used in addHeader
  struct HeaderVars {
    bytes32 entireHeaderHash;
    int score;
    address notaryAddr;
    bool isNewHead;
  }

  address[] public notaryPool;

  // Notary registry (deregistered is 0 for not yet deregistered notaries)
  mapping (address => Notary) public notaryRegistry;
  // shardId => (header_hash => tree root)
  mapping (uint => mapping (bytes32 => bytes32)) public collationTrees;
  // shardId => (headerHash => CollationHeader)
  mapping (int => mapping (bytes32 => CollationHeader)) public collationHeaders;
  // shardId => headerHash
  mapping (int => bytes32) shardead;

  // Number of notaries
  uint public notaryPoolLength;
  // Indexes of empty slots caused by the function `withdraw`
  uint[] emptySlotsStack;
  // The top index of the stack in emptySlotsStack
  uint emptySlotsStackTop;

  // Notary sample size at current period and next period
  uint currentPeriodNotarySampleSize;
  uint nextPeriodNotarySampleSize;
  uint sampleSizeLastUpdatedPeriod;

  // Constant values
  uint constant PERIOD_LENGTH = 5;
  // Exact collation body size (1mb)
  uint constant COLLATION_SIZE = 2 ** 20;
  // Number of shards
  uint constant SHARD_COUNT = 100;
  // The minimum deposit size for a notary
  uint constant NOTARY_DEPOSIT = 1000 ether;
  // The reward for notary on voting for a collation
  uint constant NOTARY_REWARD = 0.001 ether;
  // Time the ether is locked by notaries
  uint NOTARY_LOCKUP_LENGTH = 16128;
  // Number of periods ahead of current period, which the contract
  // is able to return the notary of that period
  uint constant LOOKAHEAD_LENGTH = 4;
  // Number of notaries to select from notary pool for each shard in each period
  uint constant COMMITTEE_SIZE = 135;
  // Threshold(number of notaries in committee) for a proposal to be deem accepted
  uint constant QUORUM_SIZE = 90;

  // Log the latest period number of the shard
  mapping (int => int) public periodHead;

  function SMC(uint notary_lockup_length) public {
    NOTARY_LOCKUP_LENGTH = notary_lockup_length;
  }

  function isNotaryInCommittee(uint shardId, uint period) public view returns(bool) {
    uint currentPeriod = block.number / PERIOD_LENGTH;

    // Determine notary pool length based on notary sample size
    uint sampleSize;
    if (period > sampleSizeLastUpdatedPeriod) {
      sampleSize = nextPeriodNotarySampleSize;
    } else {
      sampleSize = currentPeriodNotarySampleSize;
    }

    uint periodToLook = period * PERIOD_LENGTH - 1;

    for (uint i = 1; i <= QUORUM_SIZE; i++) {
      uint index = uint(keccak256(block.blockhash(periodToLook), index)) % sampleSize;
      if (notaryPool[index] == msg.sender) {
        return true;
      }
    }
    return false; 
  }

  function registerNotary() public payable returns(bool) {
    address notaryAddress = msg.sender;
    require(!notaryRegistry[notaryAddress].deposited);
    require(msg.value == NOTARY_DEPOSIT);

    updateNotarySampleSize();
    
    uint index;
    if (!emptyStack()) {
      index = stackPop();
      notaryPool[index] = notaryAddress;
    }
    else {
      index = notaryPoolLength; 
      notaryPool.push(notaryAddress);
    }
    ++notaryPoolLength;

    notaryRegistry[notaryAddress] = Notary({
      deregisteredPeriod: 0,
      poolIndex: index,
      deposited: true
    });

    // if current index is greater than notary sample size, increase notary sample size for next period
    if (index >= nextPeriodNotarySampleSize) {
      nextPeriodNotarySampleSize = index + 1;
    }

    NotaryRegistered(notaryAddress, index);
    return true;
  }

  function deregisterNotary() public returns(bool) {
    address notaryAddress = msg.sender;
    uint index = notaryRegistry[notaryAddress].poolIndex;
    require(notaryRegistry[notaryAddress].deposited);
    require(notaryPool[index] == notaryAddress);

    updateNotarySampleSize();
    
    uint deregisteredPeriod = block.number / PERIOD_LENGTH;
    notaryRegistry[notaryAddress].deregisteredPeriod = deregisteredPeriod;

    stackPush(index);
    --notaryPoolLength;
    NotaryDeregistered(notaryAddress, index, deregisteredPeriod);
    return true;
  }

  function releaseNotary() public returns(bool) {
    address notaryAddress = msg.sender;
    uint index = notaryRegistry[notaryAddress].poolIndex;
    require(notaryRegistry[notaryAddress].deposited == true);
    require(notaryRegistry[notaryAddress].deregisteredPeriod != 0);
    require((block.number / PERIOD_LENGTH) > (notaryRegistry[notaryAddress].deregisteredPeriod + NOTARY_LOCKUP_LENGTH));

    delete notaryRegistry[notaryAddress];
    notaryAddress.transfer(NOTARY_DEPOSIT);
    NotaryReleased(notaryAddress, index);
    return true;
  }

  function updateNotarySampleSize() private returns(bool) {
    uint currentPeriod = block.number / PERIOD_LENGTH;
    require(currentPeriod < sampleSizeLastUpdatedPeriod);

    currentPeriodNotarySampleSize = nextPeriodNotarySampleSize;
    sampleSizeLastUpdatedPeriod = currentPeriod;
    return true;
  }

  function addHeader(uint _shardId, uint period, bytes32 chunkRoot,
                     address proposerAddress) public returns(bool) {
    /*
      TODO: Anyone can call this at any time. The first header to get included for a given shard in a given period gets in,
      all others don’t. This function just emits a log
    */
  }


  function emptyStack() internal view returns(bool) {
    return emptySlotsStackTop == 0;
  }

  function stackPush(uint index) internal {
    if (emptySlotsStack.length == emptySlotsStackTop)
      emptySlotsStack.push(index);
    else
      emptySlotsStack[emptySlotsStackTop] = index;

    ++emptySlotsStackTop;
  }
  // Pop element from stack
  // Caller should check if stack is empty
  function stackPop() internal returns(uint) {
    require(emptySlotsStackTop > 1);
    --emptySlotsStackTop;
    return emptySlotsStack[emptySlotsStackTop];
  }
}

  function computeHeaderHash(uint256 shardId,
                               bytes32 parentHash, 
                               bytes32 chunkRoot, 
                               uint256 period,
                               address proposerAddress) public returns(bytes32){

  }