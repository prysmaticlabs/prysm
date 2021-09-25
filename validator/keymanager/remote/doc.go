/*
Package remote defines an implementation of an on-disk, EIP-2335 keystore.json
approach towards defining validator accounts in Prysm. A validating private key is
encrypted using a passphrase and its resulting encrypted file is stored as a
keystore.json file under a unique, human-readable, account namespace. This imported keymanager approach
relies on storing account information on-disk, making it trivial to import, backup and
list all associated accounts for a user.

Package remote defines a keymanager implementation which connects to a remote signer
server via gRPC. The connection is established via TLS using supplied paths to
certificates and key files and allows for submitting remote signing requests for
Ethereum data structures as well as retrieving the available signing public keys from
the remote server.

Remote sign requests are defined by the following protobuf schema:

 // SignRequest is a message type used by a keymanager
 // as part of Prysm's accounts implementation.
 message SignRequest {
     // 48 byte public key corresponding to an associated private key
     // being requested to sign data.
     bytes public_key = 1;

     // Raw bytes signing root the client is requesting to sign. The client is
	 // expected to determine these raw bytes from the appropriate BLS
     // signing domain as well as the signing root of the data structure
	 // the bytes represent.
     bytes signing_root = 2;
 }

Remote signing responses will contain a BLS12-381 signature along with the
status of the signing response from the remote server, signifying the
request either failed, was denied, or completed successfully.

 message SignResponse {
     enum Status {
         UNKNOWN = 0;
         SUCCEEDED = 1;
         DENIED = 2;
         FAILED = 3;
     }

     // BLS12-381 signature for the data specified in the request.
     bytes signature = 1;
 }

The remote keymanager can be customized via a keymanageropts.json file
which requires the following schema:

 {
   "remote_address": "remoteserver.com:4000", // Remote gRPC server address.
   "remote_cert": {
     "crt_path": "/home/eth2/certs/client.crt", // Client certificate path.
     "ca_crt_path": "/home/eth2/certs/ca.crt",  // Certificate authority cert path.
     "key_path": "/home/eth2/certs/client.key", // Client key path.
   }
 }
*/
package remote
