/*
Package direct defines an implementation of an on-disk, EIP-2335 keystore.json
approach towards defining validator accounts in Prysm. A validating private key is
encrypted using a passphrase and its resulting encrypted file is stored as a
keystore.json file under a unique account namespace. This direct keymanager approach
relies on storing account information on-disk, making it trivial to import, export and
list all associated accounts for a user by

EIP-2335 is a keystore format defined by https://eips.ethereum.org/EIPS/eip-2335 for
storing and defining encryption for BLS12-381 private keys, utilized by eth2.
*/
package direct
