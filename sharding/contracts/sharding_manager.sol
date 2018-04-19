pragma solidity ^0.4.23;


contract SMC {
    event HeaderAdded(uint indexed shardId, bytes32 chunkRoot, int128 period, address proposerAddress);
    event NotaryRegistered(address notary, uint poolIndex);
    event NotaryDeregistered(address notary, uint poolIndex, uint deregisteredPeriod);
    event NotaryReleased(address notary, uint poolIndex);

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

    // notaryAddress => notaryStruct
    mapping (address => Notary) public notaryRegistry;
    // shardId => (headerHash => treeRoot)
    mapping (uint => mapping (bytes32 => bytes32)) public collationTrees;
    // shardId => (headerHash => collationHeader)
    mapping (int => mapping (bytes32 => CollationHeader)) public collationHeaders;
    // shardId => headerHash
    mapping (int => bytes32) shardHead;

    // Number of notaries
    uint public notaryPoolLength;
    // Stack of empty notary slot indicies
    uint[] emptySlotsStack;
    // Top index of the stack
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
    uint constant NOTARY_LOCKUP_LENGTH = 16128;
    // Number of periods ahead of current period, which the contract
    // is able to return the notary of that period
    uint constant LOOKAHEAD_LENGTH = 4;
    // Number of notaries to select from notary pool for each shard in each period
    uint constant COMMITTEE_SIZE = 135;
    // Threshold(number of notaries in committee) for a proposal to be accepted
    uint constant QUORUM_SIZE = 90;

    // Log the latest period number of the shard
    mapping (int => int) public periodHead;

    function isNotaryInCommittee(uint shardId, uint period) public view returns(bool) {
        uint currentPeriod = block.number / PERIOD_LENGTH;

        // Determine notary pool length based on notary sample size
        uint sampleSize;
        if (period > sampleSizeLastUpdatedPeriod) {
            sampleSize = nextPeriodNotarySampleSize;
        } else {
            sampleSize = currentPeriodNotarySampleSize;
        }

      // Get the most recent block number before the period started
        uint latestBlock = period * PERIOD_LENGTH - 1;

        for (uint i = 0; i < QUORUM_SIZE; i++) {
            uint index = uint(keccak256(block.blockhash(latestBlock), index)) % sampleSize;
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
        if (emptyStack()) {
            index = notaryPoolLength; 
            notaryPool.push(notaryAddress);          
        } else {
            index = stackPop();
            notaryPool[index] = notaryAddress;
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
        
        emit NotaryRegistered(notaryAddress, index);
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
        emit NotaryDeregistered(notaryAddress, index, deregisteredPeriod);
        return true;
    }

    function releaseNotary() public returns(bool) {
        address notaryAddress = msg.sender;
        uint index = notaryRegistry[notaryAddress].poolIndex;
        require(notaryRegistry[notaryAddress].deposited == true);
        require(notaryRegistry[notaryAddress].deregisteredPeriod != 0);
        require((block.number / PERIOD_LENGTH) > 
        (notaryRegistry[notaryAddress].deregisteredPeriod + NOTARY_LOCKUP_LENGTH));

        delete notaryRegistry[notaryAddress];
        notaryAddress.transfer(NOTARY_DEPOSIT);
        emit NotaryReleased(notaryAddress, index);
        return true;
    }

    function computeHeaderHash(uint256 shardId, bytes32 parentHash, 
    bytes32 chunkRoot, uint256 period, address proposerAddress)
    public returns(bytes32) {
      /*
        TODO: Calculate the hash of the collation header from the input parameters
      */
    }

    function addHeader(uint _shardId, uint period, bytes32 chunkRoot, 
    address proposerAddress) public returns(bool) {
      /*
        TODO: Anyone can call this at any time. The first header
        to get included for a given shard in a given period gets in,
        all others don’t. This function just emits a log
      */
    }

    function updateNotarySampleSize() internal returns(bool) {
        uint currentPeriod = block.number / PERIOD_LENGTH;
        require(currentPeriod < sampleSizeLastUpdatedPeriod);

        currentPeriodNotarySampleSize = nextPeriodNotarySampleSize;
        sampleSizeLastUpdatedPeriod = currentPeriod;
        return true;
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

    function stackPop() internal returns(uint) {
        require(emptySlotsStackTop > 1);
        --emptySlotsStackTop;
        return emptySlotsStack[emptySlotsStackTop];
    }    
}