package audit

type Auditor interface {
	IncrementSuccess()
	IncrementFailure(reason string)
	Summary() SummaryReport
	Reset()
	Lock()
	Unlock()
}
