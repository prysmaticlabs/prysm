pragma solidity 0.5.0;


contract ValidatorRegistration {

    event HashChainValue(
        bytes32 indexed previousReceiptRoot,
        byte[2064] data,
        uint totalDepositcount
    );

    event ChainStart(
        bytes32 indexed receiptRoot,
        bytes[8] time
    );

    uint public constant DEPOSIT_SIZE = 32 ether;
    uint public constant DEPOSITS_FOR_CHAIN_START = 2 ** 14;
    uint public constant MIN_TOPUP_SIZE = 1 ether;
    uint public constant GWEI_PER_ETH = 10 ** 9;
    uint public constant MERKLE_TREE_DEPTH = 32;
    uint public constant SECONDS_PER_DAY = 86400;

    mapping (uint => bytes32) public receiptTree;
    uint public totalDepositCount;

    // When users wish to become a validator by moving ETH from
    // 1.0 chian to the 2.0 chain, they should call this function
    // sending along DEPOSIT_SIZE ETH and providing depositParams
    // as a simple serialize'd DepositParams object of the following
    // form: 
    // {
    //    'pubkey': 'int256',
    //    'proof_of_possession': ['int256'],
    //    'withdrawal_credentials`: 'hash32',
    //    'randao_commitment`: 'hash32'
    // }
    function deposit(
        bytes[2064] depositParams
    )
        public
        payable
    {
        uint memory index = totalDepositCount + 2 ** MERKLE_TREE_DEPTH;
        bytes[8] storage msgGweiInBytes = bytes8(msg.value);
        bytes[8] storage timeStampInBytes = bytes8(msg.value);
        bytes[2064] storage depositData = msgGweiInBytes + timeStampInBytes + depositParams;
        
        emit HashChainValue(receiptTree[1], depositParams, totalDepositCount);

        receiptTree[index] = keccak256(depositData);
        for (uint i = 0; i < MERKLE_TREE_DEPTH; i++) {
            index = index / 2;
            receiptTree[index] = keccak256(receiptTree[index * 2] + receiptTree[index * 2 + 1]);
        }

        require(
            msg.value < DEPOSIT_SIZE,
            "Deposit can't be greater than DEPOSIT_SIZE."
        );
        require(
            msg.value > MIN_TOPUP_SIZE,
            "Deposit can't be lesser than MIN_TOPUP_SIZE."
        );
        if (msg.value == DEPOSIT_SIZE) {
            totalDepositCount++;
        }

        // When ChainStart log publishes, beacon chain node initializes the chain and use timestampDayBoundry
        // as genesis time.
        if (totalDepositCount == DEPOSITS_FOR_CHAIN_START) {
            uint memory timestampDayBoundry = block.timestamp - block.timestamp % SECONDS_PER_DAY + SECONDS_PER_DAY;
            bytes[8] memory timestampDayBoundryBytes = bytes8(timestampDayBoundry);
            emit Chainstart(receiptTree[1], timestampDayBoundryBytes);
        }
    }

    function getReceiptRoot() public constant returns(bytes32) {
        return receiptTree[1];
    }
}
