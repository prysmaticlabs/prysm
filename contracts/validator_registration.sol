pragma solidity 0.4.23;

contract ValidatorRegistration {
    event ValidatorRegistered(
        bytes32 pubKey,
        uint256 withdrawalShardID,
        address withdrawalAddressbytes32,
        bytes32 randaoCommitment
    );

    mapping (bytes32 => bool) public usedPubkey;
    
    uint public constant VALIDATOR_DEPOSIT = 32 ether;

    // Validator registers by sending a transaction of 32ETH to 
    // the following deposit function. The deposit function takes in 
    // validator's public key, withdrawal shard ID (which shard
    // to send the deposit back to), withdrawal address (which address
    // to send the deposit back to) and randao commitment.
    function deposit(
        bytes32 _pubkey,
        uint _withdrawalShardID,
        address _withdrawalAddressbytes32,
        bytes32 _randaoCommitment
        )
        public payable
        {
        require(msg.value == VALIDATOR_DEPOSIT);
        require(!usedPubkey[_pubkey]);

        usedPubkey[_pubkey] = true;

        emit ValidatorRegistered(_pubkey, _withdrawalShardID, _withdrawalAddressbytes32, _randaoCommitment);
    }
}