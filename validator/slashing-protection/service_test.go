package slashingprotection

var (
	_ = Protector(&Service{})
	_ = AttestingHistoryManager(&Service{})
)
