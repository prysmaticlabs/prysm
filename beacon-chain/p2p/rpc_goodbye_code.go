package p2p

import "github.com/prysmaticlabs/prysm/beacon-chain/p2p/types"

// RPCGoodbyeCode represents goodbye code, used in sync package.
type RPCGoodbyeCode = types.SSZUint64

const (
	// Spec defined codes.
	GoodbyeCodeClientShutdown RPCGoodbyeCode = iota
	GoodbyeCodeWrongNetwork
	GoodbyeCodeGenericError

	// Teku specific codes
	GoodbyeCodeUnableToVerifyNetwork = RPCGoodbyeCode(128)

	// Lighthouse specific codes
	GoodbyeCodeTooManyPeers = RPCGoodbyeCode(129)
	GoodbyeCodeBadScore     = RPCGoodbyeCode(250)
	GoodbyeCodeBanned       = RPCGoodbyeCode(251)
)

// GoodbyeCodeMessages defines a mapping between goodbye codes and string messages.
var GoodbyeCodeMessages = map[RPCGoodbyeCode]string{
	GoodbyeCodeClientShutdown:        "client shutdown",
	GoodbyeCodeWrongNetwork:          "irrelevant network",
	GoodbyeCodeGenericError:          "fault/error",
	GoodbyeCodeUnableToVerifyNetwork: "unable to verify network",
	GoodbyeCodeTooManyPeers:          "client has too many peers",
	GoodbyeCodeBadScore:              "peer score too low",
	GoodbyeCodeBanned:                "client banned this node",
}
