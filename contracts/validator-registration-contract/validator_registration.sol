pragma solidity ^0.5.1;

import "./SafeMath.sol";
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
    // 8 is for our local test net. 2.0 spec is 2**14 == 16384 
    uint public constant DEPOSITS_FOR_CHAIN_START = 8; // 2**14
    uint public constant MIN_TOPUP_SIZE = 1 ether;
    uint public constant GWEI_PER_ETH = 10 ** 9;
    // Setting MERKLE_TREE_DEPTH to 16 instead of 32 due to gas limit
    uint public constant MERKLE_TREE_DEPTH = 16;
    uint public constant SECONDS_PER_DAY = 86400;

    mapping (uint => bytes) public receiptTree;
    uint public fullDepositCount;
    uint public totalDepositCount;

    using SafeMath for uint256;


    // When users wish to become a validator by moving ETH from
    // 1.0 chian to the 2.0 chain, they should call this function
    // sending along DEPOSIT_SIZE ETH and providing depositParams
    // as a simple serialize'd DepositParams object of the following
    // form: 
    // {
    //    'pubkey': 'uint384',
    //    'proof_of_possession': ['uint384'],
    //    'withdrawal_credentials`: 'hash32',
    //    'randao_commitment`: 'hash32'
    // }
    function deposit(
        bytes memory depositParams
    )
        public
        payable
    {
        require(
            msg.value <= DEPOSIT_SIZE,
            "Deposit can't be greater than DEPOSIT_SIZE."
        );
        require(
            msg.value >= MIN_TOPUP_SIZE,
            "Deposit can't be lesser than MIN_TOPUP_SIZE."
        );

        uint index = totalDepositCount + 2 ** MERKLE_TREE_DEPTH;
        bytes memory msgGweiInBytes8 = abi.encodePacked(uint64(msg.value/GWEI_PER_ETH));
        bytes memory timeStampInBytes8 = abi.encodePacked(uint64(block.timestamp));
        bytes memory depositData = abi.encodePacked(msgGweiInBytes8, timeStampInBytes8, depositParams);

        emit HashChainValue(receiptTree[1], depositData, totalDepositCount);

        receiptTree[index] = abi.encodePacked(keccak256(depositData));
        for (uint i = 0; i < MERKLE_TREE_DEPTH; i++) {
            index = index / 2;
            receiptTree[index] = abi.encodePacked(keccak256(abi.encodePacked(receiptTree[index * 2], receiptTree[index * 2 + 1])));
        }
        
        totalDepositCount++;
        if (msg.value == DEPOSIT_SIZE) {
            fullDepositCount++;

            // When ChainStart log publishes, beacon chain node initializes the chain and use timestampDayBoundry
            // as genesis time.
            if (fullDepositCount == DEPOSITS_FOR_CHAIN_START) {
                uint timestampDayBoundry = block.timestamp.sub(block.timestamp).mod(SECONDS_PER_DAY).add(SECONDS_PER_DAY);
                bytes memory timestampDayBoundryBytes = abi.encodePacked(uint64(timestampDayBoundry));
                emit ChainStart(receiptTree[1], timestampDayBoundryBytes);
            }
        }
    }

    function getReceiptRoot() public view returns (bytes memory) {
        return receiptTree[1];
    }
}
