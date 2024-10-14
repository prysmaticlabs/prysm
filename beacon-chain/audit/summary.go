package audit

// SummaryReport is a struct that contains the total number of successes and failures with reasons
type SummaryReport struct {
	TotalSuccesses int
	TotalFailures  int
	Failures       []Failure
}
