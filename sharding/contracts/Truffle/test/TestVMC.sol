pragma solidity ^0.4.18;

import "truffle/Assert.sol";
import "truffle/DeployedAddresses.sol";
import "../contracts/VMC.sol";

contract TestVMC { 
    function testGasCollationLimit() public {

        VMC instance = VMC(DeployedAddresses.VMC());
        uint expected = 10000000;
        Assert.equal(instance.getCollationGasLimit(), expected, "Collation Gas Limit Should be 10000000");

    }

}