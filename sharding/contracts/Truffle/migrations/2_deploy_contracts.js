var vmc = artifacts.require("./VMC.sol")

module.exports = function(deployer) {
  deployer.deploy(vmc);
};