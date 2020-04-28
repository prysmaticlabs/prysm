package evaluators

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	e2e "github.com/prysmaticlabs/prysm/endtoend/params"
	"github.com/prysmaticlabs/prysm/endtoend/types"
	"google.golang.org/grpc"
)

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
	//{
	//	name:  "",
	//	topic: "p2p_message_failed_validation_total{topic=\"/eth2/%s/beacon_aggregate_and_proof/ssz_snappy\"}",
	//	value: "0",
	//},
	{
		name:  "memory usage",
		topic: "go_memstats_alloc_bytes",
		value: 100000000,
	},
}

var metricComparisonTests = []comparisonTest{
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
	for i := 0; i < len(conns); i++ {
		response, err := http.Get(fmt.Sprintf("http://localhost:%d/metrics", e2e.TestParams.BeaconNodeMetricsPort+i))
		if err != nil {
			return errors.Wrap(err, "failed to reach prometheus metrics page")
		}
		dataInBytes, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return err
		}
		pageContent := string(dataInBytes)
		fmt.Println(pageContent)
		if err := response.Body.Close(); err != nil {
			return err
		}

		beaconClient := eth.NewNodeClient(conns[i])
		genesis, err := beaconClient.GetGenesis(context.Background(), &ptypes.Empty{})
		if err != nil {
			return err
		}

		if genesis != nil {
			return fmt.Errorf("%#x", genesis.GenesisValidatorsRoot)
		}
		for _, test := range metricLessThanTests {
			if err := metricCheckLessThan(pageContent, test.topic, test.value); err != nil {
				return errors.Wrapf(err, "failed %s check", test.name)
			}
		}
		for _, test := range metricComparisonTests {
			if err := metricCheckComparison(pageContent, test.topic1, test.topic2, test.expectedComparison); err != nil {
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
	fmt.Println(topicValue)
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
	topic1Value, err := getValueOfTopic(pageContent, topic1)
	if err != nil {
		return err
	}
	topic2Value, err := getValueOfTopic(pageContent, topic2)
	if err != nil {
		return err
	}
	fmt.Println(topic1Value)
	fmt.Println(topic2Value)
	topicComparison := float64(topic1Value) / float64(topic2Value)
	fmt.Println(topicComparison)
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
	fmt.Println(pageContent[startIdx:endOfTopic])
	// Adding 1 to skip the space after the topic name.
	startOfValue := endOfTopic + 1
	endOfValue := strings.Index(pageContent[startOfValue:], "\n")
	if endOfValue == -1 {
		return -1, fmt.Errorf("could not find next space in %s", pageContent[startOfValue:])
	}
	fmt.Println(pageContent[startOfValue : startOfValue+endOfValue])
	metricValue := pageContent[startOfValue : startOfValue+endOfValue]
	floatResult, err := strconv.ParseFloat(metricValue, 64)
	if err != nil {
		return -1, errors.Wrapf(err, "could not parse %s for int", metricValue)
	}
	return int(floatResult), nil
}
