package audit

type Auditor interface {
	IncrementSuccess()
	IncrementFailure(reason string)
	Summary() SummaryReport
	Reset()
	Lock()
	Unlock()
	RegisterSuccessCallback(callback SuccessCallback)
	RegisterFailureCallback(callback FailureCallback)
	RegisterResetCallback(callback ResetCallback)
}
