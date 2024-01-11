package helpers

import (
	"os"
	"testing"
)

// run ClearCache before each test to prevent cross-test side effects
func TestMain(m *testing.M) {
	ClearCache()
	code := m.Run()
	os.Exit(code)
}
