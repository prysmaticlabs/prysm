package grpc

// CustomErrorMetadataKey is the name of the metadata key storing additional error information.
// Metadata value is expected to be a byte-encoded JSON object.
const CustomErrorMetadataKey = "Custom-Error"

// HttpCodeMetadataKey is the key to use when setting custom HTTP status codes in gRPC metadata.
const HttpCodeMetadataKey = "X-Http-Code"
