package benchutil

import (
	"testing"
)

func TestPreGenFullBlock(t *testing.T) {
	_, err := PreGenFullBlock()
	if err != nil {
		t.Fatal(err)
	}
}

func TestPreGenState1Epoch(t *testing.T) {
	_, err := PreGenFullBlock()
	if err != nil {
		t.Fatal(err)
	}
}

func TestPreGenState2FullEpochs(t *testing.T) {
	t.Skip("To be resolved until 5119 gets in")
	_, err := PreGenFullBlock()
	if err != nil {
		t.Fatal(err)
	}
}
