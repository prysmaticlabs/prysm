pragma solidity ^0.4.19;

contract SMC {
  event HeaderAdded(uint indexed shard_id, bytes32 parent_hash,
                    bytes32 chunk_root, int128 period, int128 height,
                    address proposer_address, uint proposer_bid,
                    bytes proposer_signature);
  event NotaryRegistered(address notary, uint pool_index);
  event NotaryDeregistered(address notary, uint pool_index, uint deregistered_period);
  event NotaryReleased(address notary, uint pool_index);
  event ProposerRegistered(uint pool_index);
  event ProposerDeregistered(uint index);
  event ProposerReleased(uint index);

  struct Notary {
    uint deregistered_period;
    uint pool_index;
    bool deposited;
  }
  
  struct Proposer {
    uint deregistered_period;
    uint balances
  }

  struct CollationHeader {
    uint shard_id;            // Number of the shard ID
    bytes32 parent_hash;      // Hash of the parent collation
    bytes32 chunk_root;       // Root hash of the collation body
    uint period;              // Period which header should be included
    uint height;              // Height of the collation
    address proposer_address;
    uint proposer_bid;     
    bytes proposer_signature; 
  }

  // Packed variables to be used in addHeader
  struct HeaderVars {
    bytes32 entireHeaderHash;
    int score;
    address notaryAddr;
    bool isNewHead;
  }

  address[] public notary_pool;
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

  // Notary sample size at current period and next period
  uint current_period_notary_sample_size;
  uint next_period_notary_sample_size;
  uint sample_size_last_updated_period;

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
  // The minimum deposit size for a proposer
  uint constant PROPOSER_DEPOSIT = 1 ether;
  // The minimum balance of a proposer (for notaries)
  uint constant MIN_PROPOSER_BALANCE = 0.1 ether;
  // Time the ether is locked by notaries (Not constant for testing)
  uint NOTARY_LOCKUP_LENGTH = 16128;
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
    NOTARY_LOCKUP_LENGTH = notary_lockup_length;
    PROPOSER_LOCKUP_LENGTH = proposer_lockup_length; 
  }

  event LOG(uint L);


  function is_notary_in_committee(uint shard_id, uint period) public view returns(bool) {
    uint current_period = block.number / PERIOD_LENGTH;

    // Determine notary pool length based on notary sample size
    uint sample_size;
    if (period > sample_size_last_updated_period) {
      sample_size = next_period_notary_sample_size;
    } else {
      sample_size = current_period_notary_sample_size;
    }

    uint period_to_look = period * PERIOD_LENGTH - 1;

    for (uint i = 1; i <= QUORUM_SIZE; i++) {
      uint index = uint(keccak256(block.blockhash(period_to_look), index)) % sample_size;
      if (notary_pool[index] == msg.sender) {
        return true;
      }
    }
    return false; 
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
    require(msg.value == NOTARY_DEPOSIT);

    update_notary_sample_size();
    
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
      deregistered_period: 0,
      pool_index: index,
      deposited: true
    });

    // if current index is greater than notary_sample_size, increase notary_sample_size for next period
    if (index >= next_period_notary_sample_size) {
      next_period_notary_sample_size = index + 1;
    }

    NotaryRegistered(notary_address, index);
    return true;
  }

  function deregister_notary() public returns(bool) {
    address notary_address = msg.sender;
    uint index = notary_registry[notary_address].pool_index;
    require(notary_registry[notary_address].deposited);
    require(notary_pool[index] == notary_address);

    update_notary_sample_size();
    
    uint deregistered_period = block.number / PERIOD_LENGTH;
    notary_registry[notary_address].deregistered_period = deregistered_period;

    stack_push(index);
    --notary_pool_len;
    NotaryDeregistered(notary_address, index, deregistered_period);
    return true;
  }

  function release_notary() public returns(bool) {
    address notary_address = msg.sender;
    uint index = notary_registry[notary_address].pool_index;
    require(notary_registry[notary_address].deposited == true);
    require(notary_registry[notary_address].deregistered_period != 0);
    require((block.number / PERIOD_LENGTH) > (notary_registry[notary_address].deregistered_period + NOTARY_LOCKUP_LENGTH));

    delete notary_registry[notary_address];
    notary_address.transfer(NOTARY_DEPOSIT);
    NotaryReleased(notary_address, index);
    return true;
  }

  function update_notary_sample_size() private returns(bool) {
    uint current_period = block.number / PERIOD_LENGTH;
    require(current_period < sample_size_last_updated_period);

    current_period_notary_sample_size = next_period_notary_sample_size;
    sample_size_last_updated_period = current_period;
    return true;
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

  function addHeader(uint _shardId, uint period, bytes32 height,
                     bytes32 _parent_hash, bytes32 chunk_root,
                     address proposer_address, uint proposer_bid,
                     bytes proposer_signature) public returns(bool) {
  //  TODO: Anyone can call this at any time. The first header to get included for a given shard in a given period gets in,
  //       all others don’t. This function just emits a log
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
