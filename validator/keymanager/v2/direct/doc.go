/*
Package direct defines an implementation of an on-disk, EIP-2335 keystore.json
approach towards defining validator accounts in Prysm. A validating private key is
encrypted using a passphrase and its resulting encrypted file is stored as a
keystore.json file under a unique, human-readable, account namespace. This direct keymanager approach
relies on storing account information on-disk, making it trivial to import, export and
list all associated accounts for a user.

EIP-2335 is a keystore format defined by https://eips.ethereum.org/EIPS/eip-2335 for
storing and defining encryption for BLS12-381 private keys, utilized by eth2. This keystore.json
format is not compatible with the current keystore standard used in eth1 due to a lack of
support for KECCAK-256. Passwords utilized for key encryptions are strings of arbitrary unicode characters.
The password is first converted to its NFKD representation, stripped of control codes specified
in the EIP link above, and finally the password is UTF-8 encoded.

Accounts are stored on disk according to the following structure using human-readable
account namespaces as directories:

 wallet-dir/
   keymanageropts.json
   personally-conscious-echidna/
     keystore.json
     deposit_data.ssz
     deposit_transaction.rlp
   shy-extroverted-robin/
     keystore.json
     deposit_data.ssz
     deposit_transaction.rlp
 passwords/
   personally-conscious-echidna.pass
   shy-extroverted-robin.pass

EIP-2335 keystores are stored alongside deposit data credentials for the
created validator accounts. An additional deposit_transaction.rlp file is stored under the account,
containing a raw bytes eth1 transaction data ready to be used to submit a 32ETH deposit to the
eth2 deposit contract for a validator. Passwords are stored in a separate directory for easy unlocking
of the associated keystores by an account namespace.

This direct keymanager can be customized via a keymanageropts.json file, which has the following
JSON schema as its options:

 {
   "direct_eip_version": "EIP-2335"
 }

Currently, the only supported value for `direct_eip_version` is "EIP-2335".
*/
package direct
