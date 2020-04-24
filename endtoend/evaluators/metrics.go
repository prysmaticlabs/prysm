package evaluators

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	e2e "github.com/prysmaticlabs/prysm/endtoend/params"
	"github.com/prysmaticlabs/prysm/endtoend/types"
	"google.golang.org/grpc"
)

// NoP2PSnappyErrors uses the prometheus page to ensure there are no errors related to P2P SSZ Snappy.
var NoP2PSnappyErrors = types.Evaluator{
	Name:   "no_p2p_snappy_errors",
	Policy: func(currentEpoch uint64) bool { return true },
	Evaluation: metricsCheckEqual(
		"p2p_message_failed_validation_total{topic=\"/eth2/%s/beacon_aggregate_and_proof/ssz_snappy\"}",
		"0",
	),
}

func metricsCheckEqual(topic string, value string) func(conns ...*grpc.ClientConn) error {
	return func(conns ...*grpc.ClientConn) error {
		count := len(conns)
		for i := 0; i < count; i++ {
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
		return nil
	}
}
