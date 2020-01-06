package benchutil

import (
	"testing"
)

func TestPregenFullBlock(t *testing.T) {
	_, err := PregenFullBlock()
	if err != nil {
		t.Fatal(err)
	}
}

func TestPregenState1Epoch(t *testing.T) {
	_, err := PregenFullBlock()
	if err != nil {
		t.Fatal(err)
	}
}

func TestPreGenState2FullEpochs(t *testing.T) {
	_, err := PregenFullBlock()
	if err != nil {
		t.Fatal(err)
	}
}