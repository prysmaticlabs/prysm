pragma solidity 0.4.23;

contract ValidatorRegistration {
    event ValidatorRegistered(
        bytes32 indexed hashedPubkey,
        uint256 withdrawalShardID,
        address indexed withdrawalAddressbytes32,
        bytes32 indexed randaoCommitment
    );

    mapping (bytes32 => bool) public usedHashedPubkey;

    uint public constant VALIDATOR_DEPOSIT = 32 ether;

    // Validator registers by sending a transaction of 32ETH to
    // the following deposit function. The deposit function takes in
    // validator's public key, withdrawal shard ID (which shard
    // to send the deposit back to), withdrawal address (which address
    // to send the deposit back to) and randao commitment.
    function deposit(
        bytes _pubkey,
        uint _withdrawalShardID,
        address _withdrawalAddressbytes32,
        bytes32 _randaoCommitment
    )
        public
        payable
    {
        require(
            msg.value == VALIDATOR_DEPOSIT,
            "Incorrect validator deposit"
        );
        require(
            _pubkey.length == 48,
            "Public key is not 48 bytes"
        );

        bytes32 hashedPubkey = keccak256(abi.encodePacked(_pubkey));
        require(
            !usedHashedPubkey[hashedPubkey],
            "Public key already used"
        );

        usedHashedPubkey[hashedPubkey] = true;

        emit ValidatorRegistered(hashedPubkey, _withdrawalShardID, _withdrawalAddressbytes32, _randaoCommitment);
    }
}
