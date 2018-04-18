pragma solidity ^0.4.19;

contract SMC {
  event HeaderAdded(uint indexed shard_id, bytes32 parent_hash,
                    bytes32 chunk_root, int128 period, int128 height,
                    address proposer_address, uint proposer_bid,
                    bytes proposer_signature);
  event NotaryRegistered(address notary, uint pool_index);
  event NotaryDeregistered(address notary, uint pool_index, int128 deregistered_period);
  event NotaryReleased(address notary, uint pool_index);
  event ProposerRegistered(uint pool_index);
  event ProposerDeregistered(uint index);
  event ProposerReleased(uint index);

  // Entry in notary_registry
  struct Notary {
    // When notary asked for unregistration
    uint deregistered;
    // The notary's pool index
    uint pool_index;
    // False if notary has not deposited, true otherwise
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
    address notaryAddr;
    bool isNewHead;
  }

  // notaryId => Notaries
  // mapping (int => Notary) public notaries;
  address[] public notary_pool;
  // proposerId => Proposer
  // mapping (int => Proposer) public proposers;
  Proposer[] public proposer_pool;

  // Notary registry (deregistered is 0 for not yet deregistered notaries)
  mapping (address => Notary) public notary_registry;
  mapping (address => Proposer) public proposer_registry;
  // shard_id => (header_hash => tree root)
  mapping (uint => mapping (bytes32 => bytes32)) public collation_trees;
  // shardId => (headerHash => CollationHeader)
  mapping (int => mapping (bytes32 => CollationHeader)) public collationHeaders;
  // shardId => headerHash
  mapping (int => bytes32) shardead;

  // Number of notaries
  uint public notary_pool_len;
  // Indexes of empty slots caused by the function `withdraw`
  uint[] empty_slots_stack;
  // The top index of the stack in empty_slots_stack
  uint empty_slots_stack_top;

  // Constant values
  uint constant PERIOD_LENGTH = 5;
  // Exact collation body size (1mb)
  uint constant COLLATION_SIZE = 2 ** 20;
  // Number of shards
  uint constant SHARD_COUNT = 100;
  // The minimum deposit size for a notary
  uint constant NOTARY_DEPOSIT = 1000 ether;
  // The minimum deposit size for a proposer
  uint constant PROPOSER_DEPOSIT = 1 ether;
  // The minimum balance of a proposer (for notaries)
  uint constant MIN_PROPOSER_BALANCE = 0.1 ether;
  // Time the ether is locked by notaries (Not constant for testing)
  uint constant NOTARY_LOCKUP_LENGTH = 16128;
  // Time the ether is locked by proposers (Not constant for testing)
  uint PROPOSER_LOCKUP_LENGTH = 48;
  // Number of periods ahead of current period, which the contract
  // is able to return the notary of that period
  uint constant LOOKAHEAD_LENGTH = 4;
  // Number of notaries to select from notary pool for each shard in each period
  uint constant COMMITTEE_SIZE = 135;
  // Threshold(number of notaries in committee) for a proposal to be deem accepted
  uint constant QUORUM_SIZE = 90;

  // Log the latest period number of the shard
  mapping (int => int) public period_head;

  function SMC(uint notary_lockup_length, uint proposer_lockup_length) public {
    COLLATOR_LOCKUP_LENGTH = notary_lockup_length;
    PROPOSER_LOCKUP_LENGTH = proposer_lockup_length; 
  }

  event LOG(uint L);
  // Uses a block hash as a seed to pseudorandomly select a signer from the notary pool.
  // [TODO] Chance of being selected should be proportional to the notary's deposit.
  // Should be able to return a value for the current period or any future period up to.
  function get_eligible_notary(uint shard_id, uint period) public view returns(address) {
    uint current_period = block.number / PERIOD_LENGTH;
    uint period_to_look = ((period - LOOKAHEAD_LENGTH) * PERIOD_LENGTH);
    require(period >= current_period);
    require(period <= (current_period + LOOKAHEAD_LENGTH));
    require(notary_pool_len > 0);
    if (period <= LOOKAHEAD_LENGTH)
      period_to_look = period;
    require(period_to_look < block.number);
    return notary_pool[uint(
      keccak256(
        block.blockhash(period_to_look),
        shard_id
      )
    ) %
    notary_pool_len
    ];
  }

  function compute_header_hash(uint256 shard_id,
                               bytes32 parent_hash, 
                               bytes32 chunk_root, 
                               uint256 period,
                               address proposer_address,
                               uint256 proposer_bid) public returns(bytes32){

  }

  function register_notary() public payable returns(bool) {
    address notary_address = msg.sender;
    require(!notary_registry[notary_address].deposited);
    require(msg.value == COLLATOR_DEPOSIT);
    
    uint index;
    if (!empty_stack()) {
      index = stack_pop();
      notary_pool[index] = notary_address;
    }
    else {
      index = notary_pool_len; 
      notary_pool.push(notary_address);
    }
    ++notary_pool_len;

    notary_registry[notary_address] = Notary({
      deregistered: 0,
      pool_index: index,
      deposited: true
    });
    NotaryRegistered(notary_address, index);
    return true;
  }

  // Removes the notary from the notary pool and sets deregistered period
  function deregister_notary() public {
    address notary_address = msg.sender;
    uint index = notary_registry[notary_address].pool_index;
    // Check if notary deposited
    require(notary_registry[notary_address].deposited);
    require(notary_pool[index] == notary_address);
    
    // Deregistered period
    notary_registry[notary_address].deregistered = block.number / PERIOD_LENGTH;

    stack_push(index);
    --notary_pool_len;
    NotaryDeregistered(index);
  }

  function release_notary() public {
    address notary_address = msg.sender;
    require(notary_registry[notary_address].deposited == true);
    // Deregistered
    require(notary_registry[notary_address].deregistered != 0);
    // Locked up period
    require((block.number / PERIOD_LENGTH) > (notary_registry[notary_address].deregistered + COLLATOR_LOCKUP_LENGTH));

    delete notary_registry[notary_address];
    notary_address.transfer(COLLATOR_DEPOSIT);
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
    // headerVars.notaryAddr = getEligibleNotary(_shardId, block.number/periodLength);
    // require(headerVars.notaryAddr != 0x0);
    // require(msg.sender == headerVars.notaryAddr);

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
