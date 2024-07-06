package payloadattestation

// Requirement represents a type of requirement for payload attestation.
type Requirement int

// RequirementList defines a list of requirements.
type RequirementList []Requirement

const (
	RequireCurrentSlot Requirement = iota
	RequireMessageNotSeen
	RequireKnownPayloadStatus
	RequireValidatorInPayload
	RequireBlockRootSeen
	RequireBlockRootValid
	RequireSignatureValid
)

// GossipRequirements defines the list of requirements for gossip payload attestation messages.
var GossipRequirements = []Requirement{
	RequireCurrentSlot,
	RequireMessageNotSeen,
	RequireKnownPayloadStatus,
	RequireValidatorInPayload,
	RequireBlockRootSeen,
	RequireBlockRootValid,
	RequireSignatureValid,
}

// GossipPayloadAttestationMessageRequirements is a requirement list for gossip payload attestation messages.
var GossipPayloadAttestationMessageRequirements = RequirementList(GossipRequirements)
