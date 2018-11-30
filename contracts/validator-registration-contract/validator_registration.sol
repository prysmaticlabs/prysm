pragma solidity ^0.4.23;
pragma experimental ABIEncoderV2;


contract ValidatorRegistration {

    event HashChainValue(
        bytes indexed previousReceiptRoot,
        bytes data,
        uint totalDepositcount
    );

    event ChainStart(
        bytes indexed receiptRoot,
        bytes time
    );

    uint public constant DEPOSIT_SIZE = 32 ether;
    uint public constant DEPOSITS_FOR_CHAIN_START = 8;
    uint public constant MIN_TOPUP_SIZE = 1 ether;
    uint public constant GWEI_PER_ETH = 10 ** 9;
    // Setting MERKLE_TREE_DEPTH to 16 instead of 32 due to gas limit
    uint public constant MERKLE_TREE_DEPTH = 16;
    uint public constant SECONDS_PER_DAY = 86400;

    mapping (uint => bytes) public receiptTree;
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
        bytes depositParams
    )
        public
        payable
    {
        uint index = totalDepositCount + 2 ** MERKLE_TREE_DEPTH;
        bytes memory msgGweiInBytes = toBytes(msg.value);
        bytes memory timeStampInBytes = toBytes(block.timestamp);
        bytes memory depositData = mergeBytes( mergeBytes(msgGweiInBytes, timeStampInBytes), depositParams);

        emit HashChainValue(receiptTree[1], depositParams, totalDepositCount);

        receiptTree[index] = abi.encodePacked(keccak256(mergeBytes(receiptTree[index * 2], receiptTree[index * 2 + 1])));
        for (uint i = 0; i < MERKLE_TREE_DEPTH; i++) {
            index = index / 2;
            receiptTree[index] = mergeBytes(receiptTree[index * 2], receiptTree[index * 2 + 1]);
        }

        require(
            msg.value <= DEPOSIT_SIZE,
            "Deposit can't be greater than DEPOSIT_SIZE."
        );
        require(
            msg.value >= MIN_TOPUP_SIZE,
            "Deposit can't be lesser than MIN_TOPUP_SIZE."
        );
        if (msg.value == DEPOSIT_SIZE) {
            totalDepositCount++;
        }

        // When ChainStart log publishes, beacon chain node initializes the chain and use timestampDayBoundry
        // as genesis time.
        if (totalDepositCount == DEPOSITS_FOR_CHAIN_START) {
            uint timestampDayBoundry = block.timestamp - block.timestamp % SECONDS_PER_DAY + SECONDS_PER_DAY;
            bytes memory timestampDayBoundryBytes = toBytes(timestampDayBoundry);
            emit ChainStart(receiptTree[1], timestampDayBoundryBytes);
        }
    }

    function getReceiptRoot() public constant returns (bytes) {
        return receiptTree[1];
    }

    function toBytes(uint x) private constant returns (bytes memory) {
        bytes memory b = new bytes(32);
        assembly { mstore(add(b, 32), x) }
        return b;
    }

    function mergeBytes(bytes memory a, bytes memory b) private returns (bytes memory c) {
        // Store the length of the first array
        uint alen = a.length;
        // Store the length of BOTH arrays
        uint totallen = alen + b.length;
        // Count the loops required for array a (sets of 32 bytes)
        uint loopsa = (a.length + 31) / 32;
        // Count the loops required for array b (sets of 32 bytes)
        uint loopsb = (b.length + 31) / 32;
        assembly {
            let m := mload(0x40)
            // Load the length of both arrays to the head of the new bytes array
            mstore(m, totallen)
            // Add the contents of a to the array
            for {  let i := 0 } lt(i, loopsa) { i := add(1, i) } { mstore(add(m, mul(32, add(1, i))), mload(add(a, mul(32, add(1, i))))) }
            // Add the contents of b to the array
            for {  let i := 0 } lt(i, loopsb) { i := add(1, i) } { mstore(add(m, add(mul(32, add(1, i)), alen)), mload(add(b, mul(32, add(1, i))))) }
            mstore(0x40, add(m, add(32, totallen)))
            c := m
        }
    }
}
