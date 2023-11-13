package apimiddleware

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/events"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/r3labs/sse/v2"
)

func TestReceiveEvents(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan *sse.Event)
	w := httptest.NewRecorder()
	w.Body = &bytes.Buffer{}
	req := httptest.NewRequest("GET", "http://foo.example", &bytes.Buffer{})
	req = req.WithContext(ctx)

	go func() {
		base64Val := "Zm9v"
		data := &EventFinalizedCheckpointJson{
			Block: base64Val,
			State: base64Val,
			Epoch: "1",
		}
		bData, err := json.Marshal(data)
		require.NoError(t, err)
		msg := &sse.Event{
			Data:  bData,
			Event: []byte(events.FinalizedCheckpointTopic),
		}
		ch <- msg
		time.Sleep(time.Second)
		cancel()
	}()

	errJson := receiveEvents(ch, w, req)
	assert.Equal(t, true, errJson == nil)

	expectedEvent := `event: finalized_checkpoint
data: {"block":"0x666f6f","state":"0x666f6f","epoch":"1","execution_optimistic":false}

`
	assert.DeepEqual(t, expectedEvent, w.Body.String())
}

func TestReceiveEvents_AggregatedAtt(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan *sse.Event)
	w := httptest.NewRecorder()
	w.Body = &bytes.Buffer{}
	req := httptest.NewRequest("GET", "http://foo.example", &bytes.Buffer{})
	req = req.WithContext(ctx)

	go func() {
		base64Val := "Zm9v"
		data := AggregatedAttReceivedDataJson{
			Aggregate: &AttestationJson{
				AggregationBits: base64Val,
				Data: &AttestationDataJson{
					Slot:            "1",
					CommitteeIndex:  "1",
					BeaconBlockRoot: base64Val,
					Source:          nil,
					Target:          nil,
				},
				Signature: base64Val,
			},
		}
		bData, err := json.Marshal(data)
		require.NoError(t, err)
		msg := &sse.Event{
			Data:  bData,
			Event: []byte(events.AttestationTopic),
		}
		ch <- msg
		time.Sleep(time.Second)
		cancel()
	}()

	errJson := receiveEvents(ch, w, req)
	assert.Equal(t, true, errJson == nil)

	expectedEvent := `event: attestation
data: {"aggregation_bits":"0x666f6f","data":{"slot":"1","index":"1","beacon_block_root":"0x666f6f","source":null,"target":null},"signature":"0x666f6f"}

`
	assert.DeepEqual(t, expectedEvent, w.Body.String())
}

func TestReceiveEvents_UnaggregatedAtt(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan *sse.Event)
	w := httptest.NewRecorder()
	w.Body = &bytes.Buffer{}
	req := httptest.NewRequest("GET", "http://foo.example", &bytes.Buffer{})
	req = req.WithContext(ctx)

	go func() {
		base64Val := "Zm9v"
		data := UnaggregatedAttReceivedDataJson{
			AggregationBits: base64Val,
			Data: &AttestationDataJson{
				Slot:            "1",
				CommitteeIndex:  "1",
				BeaconBlockRoot: base64Val,
				Source:          nil,
				Target:          nil,
			},
			Signature: base64Val,
		}
		bData, err := json.Marshal(data)
		require.NoError(t, err)
		msg := &sse.Event{
			Data:  bData,
			Event: []byte(events.AttestationTopic),
		}
		ch <- msg
		time.Sleep(time.Second)
		cancel()
	}()

	errJson := receiveEvents(ch, w, req)
	assert.Equal(t, true, errJson == nil)

	expectedEvent := `event: attestation
data: {"aggregation_bits":"0x666f6f","data":{"slot":"1","index":"1","beacon_block_root":"0x666f6f","source":null,"target":null},"signature":"0x666f6f"}

`
	assert.DeepEqual(t, expectedEvent, w.Body.String())
}

func TestReceiveEvents_EventNotSupported(t *testing.T) {
	ch := make(chan *sse.Event)
	w := httptest.NewRecorder()
	w.Body = &bytes.Buffer{}
	req := httptest.NewRequest("GET", "http://foo.example", &bytes.Buffer{})

	go func() {
		msg := &sse.Event{
			Data:  []byte("foo"),
			Event: []byte("not_supported"),
		}
		ch <- msg
	}()

	errJson := receiveEvents(ch, w, req)
	require.NotNil(t, errJson)
	assert.Equal(t, "Event type 'not_supported' not supported", errJson.Msg())
}

func TestReceiveEvents_TrailingSpace(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan *sse.Event)
	w := httptest.NewRecorder()
	w.Body = &bytes.Buffer{}
	req := httptest.NewRequest("GET", "http://foo.example", &bytes.Buffer{})
	req = req.WithContext(ctx)

	go func() {
		base64Val := "Zm9v"
		data := &EventFinalizedCheckpointJson{
			Block: base64Val,
			State: base64Val,
			Epoch: "1",
		}
		bData, err := json.Marshal(data)
		require.NoError(t, err)
		msg := &sse.Event{
			Data:  bData,
			Event: []byte("finalized_checkpoint "),
		}
		ch <- msg
		time.Sleep(time.Second)
		cancel()
	}()

	errJson := receiveEvents(ch, w, req)
	assert.Equal(t, true, errJson == nil)
	assert.Equal(t, `event: finalized_checkpoint
data: {"block":"0x666f6f","state":"0x666f6f","epoch":"1","execution_optimistic":false}

`, w.Body.String())
}

func TestWriteEvent(t *testing.T) {
	base64Val := "Zm9v"
	data := &EventFinalizedCheckpointJson{
		Block: base64Val,
		State: base64Val,
		Epoch: "1",
	}
	bData, err := json.Marshal(data)
	require.NoError(t, err)
	msg := &sse.Event{
		Data:  bData,
		Event: []byte("test_event"),
	}
	w := httptest.NewRecorder()
	w.Body = &bytes.Buffer{}

	errJson := writeEvent(msg, w, &EventFinalizedCheckpointJson{})
	require.Equal(t, true, errJson == nil)
	written := w.Body.String()
	assert.Equal(t, "event: test_event\ndata: {\"block\":\"0x666f6f\",\"state\":\"0x666f6f\",\"epoch\":\"1\",\"execution_optimistic\":false}\n\n", written)
}
