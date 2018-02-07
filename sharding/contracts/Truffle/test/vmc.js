var vmc = artifacts.require("./VMC.sol");

contract('vmc', function(accounts) {
    it("Should Get collation gas limit of 10000000", function() {
        return vmc.deployed().then(function(instance) {
          return instance.getCollationGasLimit.call();
        }).then(function(limit) {
          assert.equal(limit.valueOf(), 10000000, "Collation Gas Limit is not 10000000 ");
        });
      });


})