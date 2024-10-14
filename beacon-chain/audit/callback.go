package audit

type SuccessCallback func(int)     // Callback for success event
type FailureCallback func(Failure) // Callback for failure event
type ResetCallback func()          // Callback for reset event
