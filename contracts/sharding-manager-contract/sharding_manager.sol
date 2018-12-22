pragma solidity ^0.4.23;


contract SMC {
    event HeaderAdded(uint indexed shardId, bytes32 chunkRoot, uint period, address proposerAddress);
    event AttesterRegistered(address attester, uint poolIndex);
    event AttesterDeregistered(address attester, uint poolIndex, uint deregisteredPeriod);
    event AttesterReleased(address attester, uint poolIndex);
    event VoteSubmitted(uint indexed shardId, bytes32 chunkRoot, uint period, address attesterAddress);

    struct Attester {
        uint deregisteredPeriod;
        uint poolIndex;
        uint balance;
        bool deposited;
    }

    struct CollationRecord {
        bytes32 chunkRoot;        // Root hash of the collation body
        address proposer;         // Address of the proposer
        bool isElected;           // True if the collation has reached quorum size
        bytes32 signature;          // Signature of the collation header after proposer signs
    }

    // Attester state variables
    address[] public attesterPool;
    // attesterAddress => attesterStruct
    mapping (address => Attester) public attesterRegistry;
    // number of attesters
    uint public attesterPoolLength;
    // current vote count of each shard
    // first 31 bytes are bitfield of attester's vote
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
    // Stack of empty attester slot indicies
    uint[] emptySlotsStack;
    // Top index of the stack
    uint emptySlotsStackTop;
    // Attester sample size at current period and next period
    uint currentPeriodAttesterSampleSize;
    uint nextPeriodAttesterSampleSize;
    uint sampleSizeLastUpdatedPeriod;

    // Number of shards
    // TODO: Setting default as 100. This will be a dynamic when we introduce random beacon
    uint public shardCount = 100;

    // Constant values
    // Length of challenge period for attester's proof of custody
    uint public constant CHALLENGE_PERIOD = 25;
    // Number of blocks per period
    uint constant PERIOD_LENGTH = 5;
    // The minimum deposit size for a attester
    uint constant ATTESTER_DEPOSIT = 1000 ether;
    // Time the ether is locked by attesters
    uint constant ATTESTER_LOCKUP_LENGTH = 16128;
    // Number of attesters to select from attester pool for each shard in each period
    uint constant COMMITTEE_SIZE = 135;
    // Threshold(number of attesters in committee) for a proposal to be accepted
    uint constant QUORUM_SIZE = 90;
    // Number of periods ahead of current period, which the contract
    // is able to return the attester of that period
    uint constant LOOKAHEAD_LENGTH = 4;

    /// Checks if a attester with given shard id and period has been chosen as
    /// a committee member to vote for header added on to the main chain
    function getAttesterInCommittee(uint _shardId) public view returns(address) {
        uint period = block.number / PERIOD_LENGTH;

        updateAttesterSampleSize();

        // Determine attester pool length based on attester sample size
        uint sampleSize;
        if (period > sampleSizeLastUpdatedPeriod) {
            sampleSize = nextPeriodAttesterSampleSize;
        } else {
            sampleSize = currentPeriodAttesterSampleSize;
        }

        // Get the attester pool index to concatenate with shardId and blockHash for random sample
        uint poolIndex = attesterRegistry[msg.sender].poolIndex;

        // Get the most recent block number before the period started
        uint latestBlock = period * PERIOD_LENGTH - 1;
        uint latestBlockHash = uint(block.blockhash(latestBlock));
        uint index = uint(keccak256(latestBlockHash, poolIndex, _shardId)) % sampleSize;

        return attesterPool[index];
    }

    /// Registers attester to notatery registry, locks in the attester deposit,
    /// and returns true on success
    function registerAttester() public payable {
        address attesterAddress = msg.sender;
        require(!attesterRegistry[attesterAddress].deposited);
        require(msg.value == ATTESTER_DEPOSIT);

        updateAttesterSampleSize();

        uint index;
        if (emptyStack()) {
            index = attesterPoolLength;
            attesterPool.push(attesterAddress);
        } else {
            index = stackPop();
            attesterPool[index] = attesterAddress;
        }
        ++attesterPoolLength;

        attesterRegistry[attesterAddress] = Attester({
            deregisteredPeriod: 0,
            poolIndex: index,
            balance: msg.value,
            deposited: true
        });

        // if current index is greater than attester sample size, increase attester sample size for next period
        if (index >= nextPeriodAttesterSampleSize) {
            nextPeriodAttesterSampleSize = index + 1;
        }

        emit AttesterRegistered(attesterAddress, index);
    }

    /// Deregisters attester from notatery registry, lock up period countdowns down,
    /// attester may call releaseAttester after lock up period finishses to withdraw deposit,
    /// and returns true on success
    function deregisterAttester() public {
        address attesterAddress = msg.sender;
        uint index = attesterRegistry[attesterAddress].poolIndex;
        require(attesterRegistry[attesterAddress].deposited);
        require(attesterPool[index] == attesterAddress);

        updateAttesterSampleSize();

        uint deregisteredPeriod = block.number / PERIOD_LENGTH;
        attesterRegistry[attesterAddress].deregisteredPeriod = deregisteredPeriod;

        stackPush(index); 
        delete attesterPool[index];
        --attesterPoolLength;
        emit AttesterDeregistered(attesterAddress, index, deregisteredPeriod);
    }

    /// Removes an entry from attester registry, returns deposit back to the attester,
    /// and returns true on success.
    function releaseAttester() public {
        address attesterAddress = msg.sender;
        uint index = attesterRegistry[attesterAddress].poolIndex;
        require(attesterRegistry[attesterAddress].deposited == true);
        require(attesterRegistry[attesterAddress].deregisteredPeriod != 0);
        require((block.number / PERIOD_LENGTH) > (attesterRegistry[attesterAddress].deregisteredPeriod + ATTESTER_LOCKUP_LENGTH));

        uint balance = attesterRegistry[attesterAddress].balance;
        delete attesterRegistry[attesterAddress];
        attesterAddress.transfer(balance);
        emit AttesterReleased(attesterAddress, index);
    }

    /// Add collation header to the main chain, anyone can call this function. It emits a log
    function addHeader(
        uint _shardId,
        uint _period,
        bytes32 _chunkRoot,
        bytes32 _signature
        ) public {
        require((_shardId >= 0) && (_shardId < shardCount));
        require(_period == block.number / PERIOD_LENGTH);
        require(_period > lastSubmittedCollation[_shardId]);

        updateAttesterSampleSize();

        collationRecords[_shardId][_period] = CollationRecord({
            chunkRoot: _chunkRoot,
            proposer: msg.sender,
            isElected: false,
            signature: _signature
        });

        lastSubmittedCollation[_shardId] = block.number / PERIOD_LENGTH;
        delete currentVote[_shardId];

        emit HeaderAdded(_shardId, _chunkRoot, _period, msg.sender);
    }

    /// Sampled attester can call the following funtion to submit vote,
    /// a vote log will be emitted for client to monitor
    function submitVote(
        uint _shardId,
        uint _period,
        uint _index,
        bytes32 _chunkRoot        
    ) public {
        require((_shardId >= 0) && (_shardId < shardCount));
        require(_period == block.number / PERIOD_LENGTH);
        require(_period == lastSubmittedCollation[_shardId]);
        require(_index < COMMITTEE_SIZE);
        require(_chunkRoot == collationRecords[_shardId][_period].chunkRoot);
        require(attesterRegistry[msg.sender].deposited);
        require(!hasVoted(_shardId, _index));
        require(getAttesterInCommittee(_shardId) == msg.sender);

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
    /// if a attester has casted the vote. Right shift currentVote by index 
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

    /// Save one uint into the empty slots stack for attester to use later
    function stackPush(uint _index) internal {
        if (emptySlotsStack.length == emptySlotsStackTop)
            emptySlotsStack.push(_index);
        else
            emptySlotsStack[emptySlotsStackTop] = _index;

        ++emptySlotsStackTop;
    }

    /// To keep track of attester size in between periods, we call updateAttesterSampleSize
    /// before attester registration/deregistration so correct size can be applied next period
    function updateAttesterSampleSize() internal {
        uint currentPeriod = block.number / PERIOD_LENGTH;
        if (currentPeriod < sampleSizeLastUpdatedPeriod) {
            return;
        }
        currentPeriodAttesterSampleSize = nextPeriodAttesterSampleSize;
        sampleSizeLastUpdatedPeriod = currentPeriod;
    }

    /// Get one uint out of the empty slots stack for attester index
    function stackPop() internal returns(uint) {
        require(emptySlotsStackTop > 1);
        --emptySlotsStackTop;
        return emptySlotsStack[emptySlotsStackTop];
    }

    /// Set the index bit to one, attester uses this function to cast its vote,
    /// after the attester casts its vote, we increase currentVote's count by 1
    function castVote(uint _shardId, uint _index) internal {
        uint votes = uint(currentVote[_shardId]);
        // Get the bitfield by shifting 1 to the index
        uint indexToFlag = 2 ** (255 - _index);
        // OR with currentVote to cast attester index to 1
        votes = votes | indexToFlag;
        // Update vote count
        votes++;
        currentVote[_shardId] = bytes32(votes);
    }
       
} 