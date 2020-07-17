package sync

import (
	"bytes"
	"testing"

	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestRegularSync_generateErrorResponse(t *testing.T) {
	r := &Service{
		p2p: p2ptest.NewTestP2P(t),
	}
	data, err := r.generateErrorResponse(responseCodeServerError, "something bad happened")
	require.NoError(t, err)

	buf := bytes.NewBuffer(data)
	b := make([]byte, 1)
	if _, err := buf.Read(b); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, responseCodeServerError, b[0], "The first byte was not the status code")
	msg := &pb.ErrorResponse{}
	require.NoError(t, r.p2p.Encoding().DecodeWithMaxLength(buf, msg))
	assert.Equal(t, "something bad happened", string(msg.Message), "Received the wrong message")
}
