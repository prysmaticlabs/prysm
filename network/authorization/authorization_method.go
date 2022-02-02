package authorization

// AuthorizationMethod is an authorization method such as 'Basic' or 'Bearer'.
type AuthorizationMethod uint8

const (
	// None represents no authorization method.
	None AuthorizationMethod = iota
	// Basic represents Basic Authentication.
	Basic
	// Bearer represents Bearer Authentication (token authentication).
	Bearer
)
