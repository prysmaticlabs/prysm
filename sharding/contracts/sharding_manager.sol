pragma solidity ^0.4.23;


contract SMC {
    event HeaderAdded(uint indexed shardId, bytes32 chunkRoot, uint period, address proposerAddress);
    event NotaryRegistered(address notary, uint poolIndex);
    event NotaryDeregistered(address notary, uint poolIndex, uint deregisteredPeriod);
    event NotaryReleased(address notary, uint poolIndex);

    struct Notary {
        uint deregisteredPeriod;
        uint poolIndex;
        bool deposited;
    }

    struct CollationRecord {
        bytes32 chunkRoot;        // Root hash of the collation body
        address proposer;         // Address of the proposer
        bool isElected;           // True if the collation has reached quorum size
    }

    // Notary state variables
    address[] public notaryPool;
    // notaryAddress => notaryStruct
    mapping (address => Notary) public notaryRegistry;
    // number of notaries
    uint public notaryPoolLength;
    // current vote count of each shard
    // first 31 bytes are bitfield of individual notary's vote
    // last 1 byte is total notary's vote count
    mapping (uint => bytes32) public currentVote;

    // Collation state variables
    // shardId => (period => CollationHeader), collation records been appended by proposer
    mapping (uint => mapping (uint => CollationRecord)) public collationRecords;
    // shardId => period, last period of the submitted collation header
    mapping (uint => uint) public lastSubmittedCollation;
    // shardId => period, last period of the approved collation header
    mapping (uint => uint) public lastApprovedCollation;

    // Internal help functions variables 
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

    /// Checks if a notary with given shard id and period has been chosen as
    /// a committee member to vote for header added on to the main chain
    function getNotaryInCommittee(uint shardId, uint _index) public view returns(address) {
        uint period = block.number / PERIOD_LENGTH;

        // Determine notary pool length based on notary sample size
        uint sampleSize;
        if (period > sampleSizeLastUpdatedPeriod) {
            sampleSize = nextPeriodNotarySampleSize;
        } else {
            sampleSize = currentPeriodNotarySampleSize;
        }

      // Get the most recent block number before the period started
        uint latestBlock = period * PERIOD_LENGTH - 1;
        uint latestBlockHash = uint(block.blockhash(latestBlock));
        uint index = uint(keccak256(latestBlockHash, _index, shardId)) % sampleSize;

        return notaryPool[index];
    }

    /// Registers notary to notatery registry, locks in the notary deposit,
    /// and returns true on success
    function registerNotary() public payable {
        address notaryAddress = msg.sender;
        require(!notaryRegistry[notaryAddress].deposited);
        require(msg.value == NOTARY_DEPOSIT);

        // Track the numbers of participating notaries in between periods
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
    }

    /// Deregisters notary from notatery registry, lock up period countdowns down,
    /// notary may call releaseNotary after lock up period finishses to withdraw deposit,
    /// and returns true on success
    function deregisterNotary() public {
        address notaryAddress = msg.sender;
        uint index = notaryRegistry[notaryAddress].poolIndex;
        require(notaryRegistry[notaryAddress].deposited);
        require(notaryPool[index] == notaryAddress);

        // Track the numbers of participating notaries in between periods
        updateNotarySampleSize();

        uint deregisteredPeriod = block.number / PERIOD_LENGTH;
        notaryRegistry[notaryAddress].deregisteredPeriod = deregisteredPeriod;

        stackPush(index);
        delete notaryPool[index];
        --notaryPoolLength;
        emit NotaryDeregistered(notaryAddress, index, deregisteredPeriod);
    }

    /// Removes an entry from notary registry, returns deposit back to the notary,
    /// and returns true on success.
    function releaseNotary() public {
        address notaryAddress = msg.sender;
        uint index = notaryRegistry[notaryAddress].poolIndex;
        require(notaryRegistry[notaryAddress].deposited == true);
        require(notaryRegistry[notaryAddress].deregisteredPeriod != 0);
        require((block.number / PERIOD_LENGTH) > (notaryRegistry[notaryAddress].deregisteredPeriod + NOTARY_LOCKUP_LENGTH));

        delete notaryRegistry[notaryAddress];
        notaryAddress.transfer(NOTARY_DEPOSIT);
        emit NotaryReleased(notaryAddress, index);
    }

    /// Add collation header to the main chain, anyone can call this function. It emits a log
    function addHeader(
        uint _shardId,
        uint _period,
        bytes32 _chunkRoot
        ) public {
        require((_shardId >= 0) && (_shardId < SHARD_COUNT));
        require(block.number >= PERIOD_LENGTH);
        require(_period == block.number / PERIOD_LENGTH);
        require(_period > lastSubmittedCollation[_shardId]);

        // Track the numbers of participating notaries in between periods
        updateNotarySampleSize();

        collationRecords[_shardId][_period] = CollationRecord({
            chunkRoot: _chunkRoot,
            proposer: msg.sender
        });

        lastSubmittedCollation[_shardId] = block.number / PERIOD_LENGTH;

        emit HeaderAdded(_shardId, _chunkRoot, _period, msg.sender);
    }

    /// Sampled notary can call the following funtion to submit vote,
    /// a vote log will be emitted which client will monitor
    function submitVote(
        uint _shardId,
        uint _period,
        uint _index,
        bytes32 _chunkRoot        
    ) public {
        require(notaryRegistry[msg.sender].deposited);
        require(collationRecords[_shardId][_period].chunkRoot == _chunkRoot);
        require(!hasVoted(_shardId, _index));
        require(getNotaryInCommittee(_shardId, _index) == msg.sender);

    }

    /// To keep track of notary size in between periods, we call updateNotarySampleSize
    /// before notary registration/deregistration so correct size can be applied next period
    function updateNotarySampleSize() internal {
        uint currentPeriod = block.number / PERIOD_LENGTH;
        if (currentPeriod < sampleSizeLastUpdatedPeriod) {
            return;
        }
        currentPeriodNotarySampleSize = nextPeriodNotarySampleSize;
        sampleSizeLastUpdatedPeriod = currentPeriod;
    }

    /// Check if the empty slots stack is empty
    function emptyStack() internal view returns(bool) {
        return emptySlotsStackTop == 0;
    }

    /// Save one uint into the empty slots stack for notary to use later
    function stackPush(uint index) internal {
        if (emptySlotsStack.length == emptySlotsStackTop)
            emptySlotsStack.push(index);
        else
            emptySlotsStack[emptySlotsStackTop] = index;

        ++emptySlotsStackTop;
    }

    /// Get one uint out of the empty slots stack for notary index
    function stackPop() internal returns(uint) {
        require(emptySlotsStackTop > 1);
        --emptySlotsStackTop;
        return emptySlotsStack[emptySlotsStackTop];
    }

    /// Check if a bit is set from a given byte, this function is used to check
    /// if a notary has casted the vote, return true if voted, false if not
    function hasVoted(uint _shardId, uint _index) internal returns (bool) {
        uint byteIndex = _index / 8;
        uint bitIndex = _index % 8;
        byte _byte = currentVote[_shardId][byteIndex];
        byte bitPos = byte(2 ** (7 - bitIndex));
        return (_byte & bitPos) == bitPos;
    } 
}