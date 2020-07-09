package evaluators

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	e2e "github.com/prysmaticlabs/prysm/endtoend/params"
	"github.com/prysmaticlabs/prysm/endtoend/types"
	"github.com/prysmaticlabs/prysm/shared/p2putils"
	"google.golang.org/grpc"
)

const maxMemStatsBytes = 100000000 // 1 MB.

// MetricsCheck performs a check on metrics to make sure caches are functioning, and
// overall health is good. Not checking the first epoch so the sample size isn't too small.
var MetricsCheck = types.Evaluator{
	Name:       "metrics_check_epoch_%d",
	Policy:     afterNthEpoch(0),
	Evaluation: metricsTest,
}

type equalityTest struct {
	name  string
	topic string
	value int
}

type comparisonTest struct {
	name               string
	topic1             string
	topic2             string
	expectedComparison float64
}

var metricLessThanTests = []equalityTest{
	{
		name:  "memory usage",
		topic: "go_memstats_alloc_bytes",
		value: maxMemStatsBytes,
	},
}

const (
	p2pFailValidationTopic = "p2p_message_failed_validation_total{topic=\"%s/ssz_snappy\"}"
	p2pReceivedTotalTopic  = "p2p_message_received_total{topic=\"%s/ssz_snappy\"}"
)

var metricComparisonTests = []comparisonTest{
	{
		name:               "beacon aggregate and proof",
		topic1:             fmt.Sprintf(p2pFailValidationTopic, p2p.AggregateAndProofSubnetTopicFormat),
		topic2:             fmt.Sprintf(p2pReceivedTotalTopic, p2p.AggregateAndProofSubnetTopicFormat),
		expectedComparison: 0.8,
	},
	{
		name:               "committee index 0 beacon attestation",
		topic1:             fmt.Sprintf(p2pFailValidationTopic, fmt.Sprintf(formatTopic(p2p.AttestationSubnetTopicFormat), 0)),
		topic2:             fmt.Sprintf(p2pReceivedTotalTopic, fmt.Sprintf(formatTopic(p2p.AttestationSubnetTopicFormat), 0)),
		expectedComparison: 0.15,
	},
	{
		name:               "committee index 1 beacon attestation",
		topic1:             fmt.Sprintf(p2pFailValidationTopic, fmt.Sprintf(formatTopic(p2p.AttestationSubnetTopicFormat), 1)),
		topic2:             fmt.Sprintf(p2pReceivedTotalTopic, fmt.Sprintf(formatTopic(p2p.AttestationSubnetTopicFormat), 1)),
		expectedComparison: 0.15,
	},
	{
		name:               "committee cache",
		topic1:             "committee_cache_miss",
		topic2:             "committee_cache_hit",
		expectedComparison: 0.01,
	},
	{
		name:               "hot state cache",
		topic1:             "hot_state_cache_miss",
		topic2:             "hot_state_cache_hit",
		expectedComparison: 0.01,
	},
}

func metricsTest(conns ...*grpc.ClientConn) error {
	genesis, err := eth.NewNodeClient(conns[0]).GetGenesis(context.Background(), &ptypes.Empty{})
	if err != nil {
		return err
	}
	forkDigest, err := p2putils.CreateForkDigest(time.Unix(genesis.GenesisTime.Seconds, 0), genesis.GenesisValidatorsRoot)
	if err != nil {
		return err
	}
	for i := 0; i < len(conns); i++ {
		response, err := http.Get(fmt.Sprintf("http://localhost:%d/metrics", e2e.TestParams.BeaconNodeMetricsPort+i))
		if err != nil {
			// Continue if the connection fails, regular flake.
			continue
		}
		dataInBytes, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return err
		}
		pageContent := string(dataInBytes)
		if err := response.Body.Close(); err != nil {
			return err
		}
		time.Sleep(connTimeDelay)

		chainHead, err := eth.NewBeaconChainClient(conns[i]).GetChainHead(context.Background(), &ptypes.Empty{})
		if err != nil {
			return err
		}
		timeSlot, err := getValueOfTopic(pageContent, "beacon_clock_time_slot")
		if err != nil {
			return err
		}
		if chainHead.HeadSlot != uint64(timeSlot) {
			return fmt.Errorf("expected metrics slot to equal chain head slot, expected %d, received %d", chainHead.HeadSlot, timeSlot)
		}

		for _, test := range metricLessThanTests {
			topic := test.topic
			if strings.Contains(topic, "%x") {
				topic = fmt.Sprintf(topic, forkDigest)
			}
			if err := metricCheckLessThan(pageContent, topic, test.value); err != nil {
				return errors.Wrapf(err, "failed %s check", test.name)
			}
		}
		for _, test := range metricComparisonTests {
			topic1 := test.topic1
			if strings.Contains(topic1, "%x") {
				topic1 = fmt.Sprintf(topic1, forkDigest)
			}
			topic2 := test.topic2
			if strings.Contains(topic2, "%x") {
				topic2 = fmt.Sprintf(topic2, forkDigest)
			}
			if err := metricCheckComparison(pageContent, topic1, topic2, test.expectedComparison); err != nil {
				return err
			}
		}
	}
	return nil
}

func metricCheckLessThan(pageContent string, topic string, value int) error {
	topicValue, err := getValueOfTopic(pageContent, topic)
	if err != nil {
		return err
	}
	if topicValue >= value {
		return fmt.Errorf(
			"unexpected result for metric %s, expected less than %d, received %d",
			topic,
			value,
			topicValue,
		)
	}
	return nil
}

func metricCheckComparison(pageContent string, topic1 string, topic2 string, comparison float64) error {
	topic2Value, err := getValueOfTopic(pageContent, topic2)
	// If we can't find the first topic (error metrics), then assume the test passes.
	if topic2Value != -1 {
		return nil
	}
	if err != nil {
		return err
	}
	topic1Value, err := getValueOfTopic(pageContent, topic1)
	if topic1Value != -1 {
		return nil
	}
	if err != nil {
		return err
	}
	topicComparison := float64(topic1Value) / float64(topic2Value)
	if topicComparison >= comparison {
		return fmt.Errorf(
			"unexpected result for comparison between metric %s and metric %s, expected comparison to be %.2f, received %.2f",
			topic1,
			topic2,
			comparison,
			topicComparison,
		)
	}
	return nil
}

func getValueOfTopic(pageContent string, topic string) (int, error) {
	// Adding a space to search exactly.
	startIdx := strings.LastIndex(pageContent, topic+" ")
	if startIdx == -1 {
		return -1, fmt.Errorf("did not find requested text %s in %s", topic, pageContent)
	}
	endOfTopic := startIdx + len(topic)
	// Adding 1 to skip the space after the topic name.
	startOfValue := endOfTopic + 1
	endOfValue := strings.Index(pageContent[startOfValue:], "\n")
	if endOfValue == -1 {
		return -1, fmt.Errorf("could not find next space in %s", pageContent[startOfValue:])
	}
	metricValue := pageContent[startOfValue : startOfValue+endOfValue]
	floatResult, err := strconv.ParseFloat(metricValue, 64)
	if err != nil {
		return -1, errors.Wrapf(err, "could not parse %s for int", metricValue)
	}
	return int(floatResult), nil
}

func formatTopic(topic string) string {
	startIndex := strings.Index(topic, "%x")
	if startIndex == -1 {
		return topic
	}
	return topic[:startIndex] + "%" + topic[startIndex:]
}
