package evaluators

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	e2e "github.com/prysmaticlabs/prysm/endtoend/params"
	"github.com/prysmaticlabs/prysm/endtoend/types"
	"google.golang.org/grpc"
)

type equalityTest struct {
	name  string
	topic string
	value string
}

type comparisonTest struct {
	name               string
	topic1             string
	topic2             string
	expectedComparison float64
}

var metricEqualityTests = []equalityTest{
	{
		name:  "",
		topic: "p2p_message_failed_validation_total{topic=\"/eth2/%s/beacon_aggregate_and_proof/ssz_snappy\"}",
		value: "0",
	},
}

var metricComparisonTests = []comparisonTest{
	{
		name:               "",
		topic1:             "",
		topic2:             "",
		expectedComparison: 0.1,
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
		if err := response.Body.Close(); err != nil {
			return err
		}
		for _, test := range metricEqualityTests {
			if err := metricCheckEqual(pageContent, test.topic, test.value); err != nil {
				return err
			}
		}
		for _, test := range metricComparisonTests {
			if err := metricCheckEqual(pageContent); err != nil {
				return err
			}
		}
	}
	return nil
}

func metricCheckEqual(pageContent string, topic string, value string) error {
	searchText := topic
	fmt.Println(pageContent)
	startIdx := strings.Index(pageContent, searchText)
	if startIdx == -1 {
		return fmt.Errorf("did not find requested text in %s", pageContent)
	}
	startIdx += len(searchText)
	endIdx := strings.Index(pageContent[startIdx+1:], " ")
	metricCount := pageContent[startIdx:endIdx]
	fmt.Println(metricCount)
	if metricCount != value {
		return fmt.Errorf(
			"unexpected result for metric %s, expected equal to %s, received %s",
			topic,
			value,
			metricCount,
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
	topicComparison := topic1Value / topic2Value
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

func getValueOfTopic(pageContent string, topic string) (float64, error) {
	startIdx := strings.Index(pageContent, topic)
	if startIdx == -1 {
		return 0, fmt.Errorf("did not find requested text %s in %s", topic, pageContent)
	}
	startIdx += len(topic)
	endIdx := strings.Index(pageContent[startIdx+1:], " ")
	metricValue := pageContent[startIdx:endIdx]
	intResult, err := strconv.Atoi(metricValue)
	if err != nil {
		return 0, errors.Wrapf(err, "could not parse %s for int", metricValue)
	}
	return float64(intResult), nil
}
