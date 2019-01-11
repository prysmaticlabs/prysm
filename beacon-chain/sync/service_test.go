package sync

import "testing"

func TestStatus(t *testing.T) {
	fStatus := ss.failStatus
	sStatus := ss.Status()
	if err := sStatus; err != fStatus {
		t.Errorf("Expected match of ss.Status() and ss.failStatus, but got %v, %v", sStatus, fStatus)
	}
}