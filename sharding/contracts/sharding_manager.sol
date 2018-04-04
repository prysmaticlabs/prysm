pragma solidity ^0.4.19;

contract SMC {
  event HeaderAdded(uint indexed shard_id, bytes32 parent_hash,
                    bytes32 chunk_root, int128 period, int128 height,
                    address proposer_address, uint proposer_bid,
                    bytes proposer_signature);
  event CollatorRegistered(address collator, uint pool_index);
  event CollatorDeregistered(uint pool_index);
  event CollatorReleased(uint pool_index);
  event ProposerRegistered(uint pool_index);
  event ProposerDeregistered(uint index);
  event ProposerReleased(uint index);

  // Entry in collator_registry
  struct Collator {
    // When collator asked for unregistration
    uint deregistered;
    // The collator's pool index
    uint pool_index;
    // False if collator has not deposited, true otherwise
    bool deposited;
  }
  
  // Entry in proposer_registry
  struct Proposer {
    // Shard id of the deposit
    uint shardId;
    // Deposit in ether in each shard
    uint[] deposit;
  }

  struct CollationHeader {
    uint256 shard_id;         // pointer to shard
    bytes32 parent_hash;      // pointer to parent header
    bytes32 chunk_root;       // pointer to collation body
    int128 period;            // Period which header should be included
    int128 height;            // Collation's height
    address proposer_address; // Proposer's address 
    uint256 proposer_bid;     // Proposer's bid
    bytes proposer_signature; // Proposer's signature
  }

  // Packed variables to be used in addHeader
  struct HeaderVars {
    bytes32 entireHeaderHash;
    int score;
    address collatorAddr;
    bool isNewHead;
  }

  // collatorId => Collators
  // mapping (int => Collator) public collators;
  address[] public collator_pool;
  // proposerId => Proposer
  // mapping (int => Proposer) public proposers;
  Proposer[] public proposer_pool;

  // Collator registry (deregistered is 0 for not yet deregistered collators)
  mapping (address => Collator) public collator_registry;
  mapping (address => Proposer) public proposer_registry;
  // shard_id => (header_hash => tree root)
  mapping (uint => mapping (bytes32 => bytes32)) public collation_trees;
  // shardId => (headerHash => CollationHeader)
  mapping (int => mapping (bytes32 => CollationHeader)) public collationHeaders;
  // shardId => headerHash
  mapping (int => bytes32) shardead;

  // Number of collators
  uint public collator_pool_len;
  // Indexes of empty slots caused by the function `withdraw`
  uint[] empty_slots_stack;
  // The top index of the stack in empty_slots_stack
  uint empty_slots_stack_top;

  // Constant values
  uint constant PERIOD_LENGTH = 5;
  // Exact collation body size (1mb)
  uint constant COLLATION_SIZE = 2 ** 20;
  // Subsidy in vEth
  uint constant COLLATOR_SUBSIDY = 0.001 ether;
  // Number of shards
  uint constant SHARD_COUNT = 100;
  // The minimum deposit size for a collator
  uint constant COLLATOR_DEPOSIT = 1000 ether;
  // The minimum deposit size for a proposer
  uint constant PROPOSER_DEPOSIT = 1 ether;
  // The minimum balance of a proposer (for collators)
  uint constant MIN_PROPOSER_BALANCE = 0.1 ether;
  // Time the ether is locked by collators (Not constant for testing)
  uint COLLATOR_LOCKUP_LENGTH = 16128;
  // Time the ether is locked by proposers (Not constant for testing)
  uint PROPOSER_LOCKUP_LENGTH = 48;
  // Number of periods ahead of current period, which the contract
  // is able to return the collator of that period
  uint constant LOOKAHEAD_LENGTH = 4;

  // Log the latest period number of the shard
  mapping (int => int) public period_head;

  function SMC(uint collator_lockup_length, uint proposer_lockup_length) public {
    COLLATOR_LOCKUP_LENGTH = collator_lockup_length;
    PROPOSER_LOCKUP_LENGTH = proposer_lockup_length; 
  }

  event LOG(uint L);
  // Uses a block hash as a seed to pseudorandomly select a signer from the collator pool.
  // [TODO] Chance of being selected should be proportional to the collator's deposit.
  // Should be able to return a value for the current period or any future period up to.
  function get_eligible_collator(uint shard_id, uint period) public view returns(address) {
    uint current_period = block.number / PERIOD_LENGTH;
    uint period_to_look = ((period - LOOKAHEAD_LENGTH) * PERIOD_LENGTH);
    require(period >= current_period);
    require(period <= (current_period + LOOKAHEAD_LENGTH));
    require(collator_pool_len > 0);
    if (period <= LOOKAHEAD_LENGTH)
      period_to_look = period;
    require(period_to_look < block.number);
    return collator_pool[uint(
      keccak256(
        block.blockhash(period_to_look),
        shard_id
      )
    ) %
    collator_pool_len
    ];
  }

  function compute_header_hash(uint256 shard_id,
                               bytes32 parent_hash, 
                               bytes32 chunk_root, 
                               uint256 period,
                               address proposer_address,
                               uint256 proposer_bid) public returns(bytes32){

  }

  function register_collator() public payable returns(bool) {
    address collator_address = msg.sender;
    require(!collator_registry[collator_address].deposited);
    require(msg.value == COLLATOR_DEPOSIT);
    
    uint index;
    if (!empty_stack()) {
      index = stack_pop();
      collator_pool[index] = collator_address;
    }
    else {
      index = collator_pool_len; 
      collator_pool.push(collator_address);
    }
    ++collator_pool_len;

    collator_registry[collator_address] = Collator({
      deregistered: 0,
      pool_index: index,
      deposited: true
    });
    CollatorRegistered(collator_address, index);
    return true;
  }

  // Removes the collator from the collator pool and sets deregistered period
  function deregister_collator() public {
    address collator_address = msg.sender;
    uint index = collator_registry[collator_address].pool_index;
    // Check if collator deposited
    require(collator_registry[collator_address].deposited);
    require(collator_pool[index] == collator_address);
    
    // Deregistered period
    collator_registry[collator_address].deregistered = block.number / PERIOD_LENGTH;

    stack_push(index);
    --collator_pool_len;
    CollatorDeregistered(index);
  }

  function release_collator() public {
    address collator_address = msg.sender;
    require(collator_registry[collator_address].deposited == true);
    // Deregistered
    require(collator_registry[collator_address].deregistered != 0);
    // Locked up period
    require((block.number / PERIOD_LENGTH) > (collator_registry[collator_address].deregistered + COLLATOR_LOCKUP_LENGTH));

    delete collator_registry[collator_address];
    collator_address.transfer(COLLATOR_DEPOSIT);
  }

  function register_proposer() public payable returns(int) {

  }

  function deregister_proposer() public {

  }

  function release_proposer() public {

  }

  function proposer_add_balance(uint shard_id) payable public {

  }

  function proposer_withdraw_balance(uint shard_id) public {

  }

  // Attempts to process a collation header, returns true on success, reverts on failure.
  function addHeader(uint _shardId, uint period, bytes32 height,
                     bytes32 _parent_hash, bytes32 chunk_root,
                     address proposer_address, uint proposer_bid,
                     bytes proposer_signature) public returns(bool) {
    // HeaderVars memory headerVars;

    // // Check if the header is valid
    // require((_shardId >= 0) && (_shardId < shardCount));
    // require(block.number >= periodLength);
    // require(_expectedPeriodNumber == block.number / periodLength);
    // require(_periodStartPrevHash == block.blockhash(_expectedPeriodNumber * periodLength - 1));

    // // Check if this header already exists
    // headerVars.entireHeaderHash = keccak256(_shardId, _expectedPeriodNumber, _periodStartPrevHash,
    //                                _parentHash, _transactionRoot, bytes32(_coinbase),
    //                                _stateRoot, _receiptRoot, _number);
    // assert(collationHeaders[_shardId][headerVars.entireHeaderHash].score == 0);
    // // Check whether the parent exists.
    // // if (parent_collation_hash == 0), i.e., is the genesis,
    // // then there is no need to check.
    // if (_parentHash != 0x0)
    //     assert(collationHeaders[_shardId][_parentHash].score > 0);
    // // Check if only one collation in one period
    // assert(periodHead[_shardId] < int(_expectedPeriodNumber));

    // // Check the signature with validation_code_addr
    // headerVars.collatorAddr = getEligibleCollator(_shardId, block.number/periodLength);
    // require(headerVars.collatorAddr != 0x0);
    // require(msg.sender == headerVars.collatorAddr);

    // // Check score == collationNumber
    // headerVars.score = collationHeaders[_shardId][_parentHash].score + 1;
    // require(_number == headerVars.score);

    // // Add the header
    // collationHeaders[_shardId][headerVars.entireHeaderHash] = CollationHeader({
    //   parentHash: _parentHash,
    //   score: headerVars.score
    // });

    // // Update the latest period number
    // periodHead[_shardId] = int(_expectedPeriodNumber);

    // // Determine the head
    // if (headerVars.score > collationHeaders[_shardId][shardHead[_shardId]].score) {
    //   shardHead[_shardId] = headerVars.entireHeaderHash;
    //   headerVars.isNewHead = true;
    // }

    // CollationAdded(_shardId, _expectedPeriodNumber, _periodStartPrevHash,
    //                _parentHash, _transactionRoot, _coinbase, _stateRoot, 
    //                _receiptRoot, _number, headerVars.isNewHead, headerVars.score);

    // return true;
  }


  function empty_stack() internal view returns(bool) {
    return empty_slots_stack_top == 0;
  }

  function stack_push(uint index) internal {
    if (empty_slots_stack.length == empty_slots_stack_top)
      empty_slots_stack.push(index);
    else
      empty_slots_stack[empty_slots_stack_top] = index;

    ++empty_slots_stack_top;
  }
  // Pop element from stack
  // Caller should check if stack is empty
  function stack_pop() internal returns(uint) {
    require(empty_slots_stack_top > 1);
    --empty_slots_stack_top;
    return empty_slots_stack[empty_slots_stack_top];
  }
}
