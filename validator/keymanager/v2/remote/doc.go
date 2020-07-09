/*
Package direct defines an implementation of an on-disk, EIP-2335 keystore.json
approach towards defining validator accounts in Prysm. A validating private key is
encrypted using a passphrase and its resulting encrypted file is stored as a
keystore.json file under a unique, human-readable, account namespace. This direct keymanager approach
relies on storing account information on-disk, making it trivial to import, export and
list all associated accounts for a user.

Package remote defines a keymanager implementation which connects to a remote signer
server via gRPC. The connection is established via TLS using supplied paths to
certificates and key files and allows for submitting remote signing requests for
eth2 data structures as well as retrieving the available signing public keys from
the remote server.

Remote sign requests are defined by the following protobuf schema:

 // SignRequest is a message type used by a keymanager
 // as part of Prysm's accounts implementation.
 message SignRequest {
     // 48 byte public key corresponding to an associated private key
     // being requested to sign data.
     bytes public_key = 1;

     // Raw bytes data the client is requesting to sign.
     bytes data = 2;

     // Signature domain required for BLS signing.
     bytes signature_domain = 3;

     // The raw object being signed by the request.
     oneof object {
         ethereum.eth.v1alpha1.BeaconBlock block = 4;
         ethereum.eth.v1alpha1.AttestationData attestation_data = 5;
         ethereum.eth.v1alpha1.AggregateAttestationAndProof
            aggregate_attestation_and_proof = 6;
         ethereum.eth.v1alpha1.VoluntaryExit exit = 7;
         uint64 slot = 8;
     }
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
