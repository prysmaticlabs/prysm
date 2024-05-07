package experiment

import "testing"

func TestProcessAltair(t *testing.T) {
	body := &AltairBody{}
	block := &Block[*AltairBody]{body: body}
	processAltair[BlockHasBody[*AltairBody], *AltairBody](block)
}
