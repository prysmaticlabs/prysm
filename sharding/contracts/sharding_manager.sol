pragma solidity ^0.4.23;


contract SMC {
    event HeaderAdded(uint indexed shardId, bytes32 chunkRoot, uint period, address proposerAddress);
    event NotaryRegistered(address notary, uint poolIndex);
    event NotaryDeregistered(address notary, uint poolIndex, uint deregisteredPeriod);
    event NotaryReleased(address notary, uint poolIndex);
    event VoteSubmitted(uint indexed shardId, bytes32 chunkRoot, uint period, address notaryAddress);

    struct Notary {
        uint deregisteredPeriod;
        uint poolIndex;
        uint balance;
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
    // first 31 bytes are bitfield of notary's vote
    // last 1 byte is the number of the total votes
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
    // Length of challenge period for notary's proof of custody
    uint public constant CHALLENGE_PERIOD = 25;
    // Number of blocks per period
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
    function getNotaryInCommittee(uint _shardId) public view returns(address) {
        uint period = block.number / PERIOD_LENGTH;

        updateNotarySampleSize();

        // Determine notary pool length based on notary sample size
        uint sampleSize;
        if (period > sampleSizeLastUpdatedPeriod) {
            sampleSize = nextPeriodNotarySampleSize;
        } else {
            sampleSize = currentPeriodNotarySampleSize;
        }

        // Get the notary pool index to concatenate with shardId and blockHash for random sample
        uint poolIndex = notaryRegistry[msg.sender].poolIndex;

        // Get the most recent block number before the period started
        uint latestBlock = period * PERIOD_LENGTH - 1;
        uint latestBlockHash = uint(block.blockhash(latestBlock));
        uint index = uint(keccak256(latestBlockHash, poolIndex, _shardId)) % sampleSize;

        return notaryPool[index];
    }

    /// Registers notary to notatery registry, locks in the notary deposit,
    /// and returns true on success
    function registerNotary() public payable {
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
            balance: msg.value,
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

        uint balance = notaryRegistry[notaryAddress].balance;
        delete notaryRegistry[notaryAddress];
        notaryAddress.transfer(balance);
        emit NotaryReleased(notaryAddress, index);
    }

    /// Add collation header to the main chain, anyone can call this function. It emits a log
    function addHeader(
        uint _shardId,
        uint _period,
        bytes32 _chunkRoot
        ) public {
        require((_shardId >= 0) && (_shardId < SHARD_COUNT));
        require(_period == block.number / PERIOD_LENGTH);
        require(_period > lastSubmittedCollation[_shardId]);

        updateNotarySampleSize();

        collationRecords[_shardId][_period] = CollationRecord({
            chunkRoot: _chunkRoot,
            proposer: msg.sender,
            isElected: false
        });

        lastSubmittedCollation[_shardId] = block.number / PERIOD_LENGTH;
        delete currentVote[_shardId];

        emit HeaderAdded(_shardId, _chunkRoot, _period, msg.sender);
    }

    /// Sampled notary can call the following funtion to submit vote,
    /// a vote log will be emitted for client to monitor
    function submitVote(
        uint _shardId,
        uint _period,
        uint _index,
        bytes32 _chunkRoot        
    ) public {
        require((_shardId >= 0) && (_shardId < SHARD_COUNT));
        require(_period == block.number / PERIOD_LENGTH);
        require(_period == lastSubmittedCollation[_shardId]);
        require(_index < COMMITTEE_SIZE);
        require(_chunkRoot == collationRecords[_shardId][_period].chunkRoot);
        require(notaryRegistry[msg.sender].deposited);
        require(!hasVoted(_shardId, _index));
        require(getNotaryInCommittee(_shardId) == msg.sender);

        castVote(_shardId, _index);
        uint voteCount = getVoteCount(_shardId);
        if (voteCount >= QUORUM_SIZE) {
            lastApprovedCollation[_shardId] = _period;
            collationRecords[_shardId][_period].isElected = true;
        }
        emit VoteSubmitted(_shardId, _chunkRoot, _period, msg.sender);
    }

    /// Returns total vote count of currentVote
    /// the vote count is stored in the last byte of currentVote 
    function getVoteCount(uint _shardId) public view returns (uint) {
        uint votes = uint(currentVote[_shardId]);
        // Extra the last byte of currentVote
        return votes % 256;
    }

    /// Check if a bit is set, this function is used to check
    /// if a notary has casted the vote. Right shift currentVote by index 
    /// and AND with 1, return true if voted, false if not
    function hasVoted(uint _shardId, uint _index) public view returns (bool) {
        uint votes = uint(currentVote[_shardId]);
        // Shift currentVote to right by given index 
        votes = votes >> (255 - _index);
        // AND 1 to neglect everything but bit 0, then compare to 1
        return votes & 1 == 1;
    }

    /// Check if the empty slots stack is empty
    function emptyStack() internal view returns(bool) {
        return emptySlotsStackTop == 0;
    }

    /// Save one uint into the empty slots stack for notary to use later
    function stackPush(uint _index) internal {
        if (emptySlotsStack.length == emptySlotsStackTop)
            emptySlotsStack.push(_index);
        else
            emptySlotsStack[emptySlotsStackTop] = _index;

        ++emptySlotsStackTop;
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

    /// Get one uint out of the empty slots stack for notary index
    function stackPop() internal returns(uint) {
        require(emptySlotsStackTop > 1);
        --emptySlotsStackTop;
        return emptySlotsStack[emptySlotsStackTop];
    }

    /// Set the index bit to one, notary uses this function to cast its vote,
    /// after the notary casts its vote, we increase currentVote's count by 1
    function castVote(uint _shardId, uint _index) internal {
        uint votes = uint(currentVote[_shardId]);
        // Get the bitfield by shifting 1 to the index
        uint indexToFlag = 2 ** (255 - _index);
        // OR with currentVote to cast notary index to 1
        votes = votes | indexToFlag;
        // Update vote count
        votes++;
        currentVote[_shardId] = bytes32(votes);
    }
       
} 