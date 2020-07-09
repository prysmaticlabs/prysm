/*
Package direct defines an implementation of an on-disk, EIP-2335 keystore.json
approach towards defining validator accounts in Prysm. A validating private key is
encrypted using a passphrase and its resulting encrypted file is stored as a
keystore.json file under a unique, human-readable, account namespace. This direct keymanager approach
relies on storing account information on-disk, making it trivial to import, export and
list all associated accounts for a user.

Package remote

The remote key manager connects to a walletd instance.  The options are:
  - location This is the location to look for wallets.  If not supplied it will
    use the standard (operating system-dependent) path.
  - accounts This is a list of account specifiers.  An account specifier is of
    the form <wallet name>/[account name],  where the account name can be a
    regular expression.  If the account specifier is just <wallet name> all
    accounts in that wallet will be used.  Multiple account specifiers can be
    supplied if required.
  - certificates This provides paths to certificates:
    - ca_cert This is the path to the server's certificate authority certificate file
    - client_cert This is the path to the client's certificate file
    - client_key This is the path to the client's key file

An sample keymanager options file (with annotations; these should be removed if
using this as a template) is:

  {
	"location":    "host.example.com:12345", // Connect to walletd at host.example.com on port 12345
    "accounts":    ["Validators/Account.*"]  // Use all accounts in the 'Validators' wallet starting with 'Account'
	"certificates": {
	  "ca_cert": "/home/eth2/certs/ca.crt"         // Certificate file for the CA that signed the server's certificate
	  "client_cert": "/home/eth2/certs/client.crt" // Certificate file for this client
	  "client_key": "/home/eth2/certs/client.key"  // Key file for this client
	}
  }`
 {
   direct_eip_version: string
 }
*/
package remote
