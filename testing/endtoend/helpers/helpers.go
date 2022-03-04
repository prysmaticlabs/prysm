// Package helpers defines helper functions to peer into
// end to end processes and kill processes as needed.
package helpers

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/config/params"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	e2e "github.com/prysmaticlabs/prysm/testing/endtoend/params"
	e2etypes "github.com/prysmaticlabs/prysm/testing/endtoend/types"
	"github.com/prysmaticlabs/prysm/time/slots"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

const (
	maxPollingWaitTime    = 60 * time.Second // A minute so timing out doesn't take very long.
	filePollingInterval   = 500 * time.Millisecond
	memoryHeapFileName    = "node_heap_%d.pb.gz"
	cpuProfileFileName    = "node_cpu_profile_%d.pb.gz"
	fileBufferSize        = 64 * 1024
	maxFileBufferSize     = 1024 * 1024
	AltairE2EForkEpoch    = 6
	BellatrixE2EForkEpoch = 8
)

// Graffiti is a list of sample graffiti strings.
var Graffiti = []string{"Sushi", "Ramen", "Takoyaki"}

// DeleteAndCreateFile checks if the file path given exists, if it does, it deletes it and creates a new file.
// If not, it just creates the requested file.
func DeleteAndCreateFile(tmpPath, fileName string) (*os.File, error) {
	filePath := path.Join(tmpPath, fileName)
	if _, err := os.Stat(filePath); os.IsExist(err) {
		if err = os.Remove(filePath); err != nil {
			return nil, err
		}
	}

	newFile, err := os.Create(filepath.Clean(path.Join(tmpPath, fileName)))

	if err != nil {
		return nil, err
	}
	return newFile, nil
}

// WaitForTextInFile checks a file every polling interval for the text requested.
func WaitForTextInFile(file *os.File, text string) error {
	d := time.Now().Add(maxPollingWaitTime)
	ctx, cancel := context.WithDeadline(context.Background(), d)
	defer cancel()

	// Use a ticker with a deadline to poll a given file.
	ticker := time.NewTicker(filePollingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			contents, err := ioutil.ReadAll(file)
			if err != nil {
				return err
			}
			return fmt.Errorf("could not find requested text \"%s\" in logs:\n%s", text, contents)
		case <-ticker.C:
			fileScanner := bufio.NewScanner(file)
			buf := make([]byte, 0, fileBufferSize)
			fileScanner.Buffer(buf, maxFileBufferSize)
			for fileScanner.Scan() {
				scanned := fileScanner.Text()
				if strings.Contains(scanned, text) {
					return nil
				}
			}
			if err := fileScanner.Err(); err != nil {
				return err
			}
			_, err := file.Seek(0, io.SeekStart)
			if err != nil {
				return err
			}
		}
	}
}

// FindFollowingTextInFile checks a file every polling interval for the  following text requested.
func FindFollowingTextInFile(file *os.File, text string) (string, error) {
	d := time.Now().Add(maxPollingWaitTime)
	ctx, cancel := context.WithDeadline(context.Background(), d)
	defer cancel()

	// Use a ticker with a deadline to poll a given file.
	ticker := time.NewTicker(filePollingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			contents, err := ioutil.ReadAll(file)
			if err != nil {
				return "", err
			}
			return "", fmt.Errorf("could not find requested text \"%s\" in logs:\n%s", text, contents)
		case <-ticker.C:
			fileScanner := bufio.NewScanner(file)
			buf := make([]byte, 0, fileBufferSize)
			fileScanner.Buffer(buf, maxFileBufferSize)
			for fileScanner.Scan() {
				scanned := fileScanner.Text()
				if strings.Contains(scanned, text) {
					lastIdx := strings.LastIndex(scanned, text)
					truncatedIdx := lastIdx + len(text)
					if len(scanned) <= truncatedIdx {
						return "", fmt.Errorf("truncated index is larger than the size of whole scanned line")
					}
					splitObjs := strings.Split(scanned[truncatedIdx:], " ")
					if len(splitObjs) == 0 {
						return "", fmt.Errorf("0 split substrings retrieved")
					}
					return splitObjs[0], nil
				}
			}
			if err := fileScanner.Err(); err != nil {
				return "", err
			}
			_, err := file.Seek(0, io.SeekStart)
			if err != nil {
				return "", err
			}
		}
	}
}

// GraffitiYamlFile outputs graffiti YAML file into a testing directory.
func GraffitiYamlFile(testDir string) (string, error) {
	b := []byte(`default: "Rice"
random: 
  - "Sushi"
  - "Ramen"
  - "Takoyaki"
`)
	f := filepath.Join(testDir, "graffiti.yaml")
	if err := ioutil.WriteFile(f, b, os.ModePerm); err != nil {
		return "", err
	}
	return f, nil
}

// LogOutput logs the output of all log files made.
func LogOutput(t *testing.T) {
	// Log out errors from beacon chain nodes.
	for i := 0; i < e2e.TestParams.BeaconNodeCount; i++ {
		beaconLogFile, err := os.Open(path.Join(e2e.TestParams.LogPath, fmt.Sprintf(e2e.BeaconNodeLogFileName, i)))
		if err != nil {
			t.Fatal(err)
		}
		LogErrorOutput(t, beaconLogFile, "beacon chain node", i)

		validatorLogFile, err := os.Open(path.Join(e2e.TestParams.LogPath, fmt.Sprintf(e2e.ValidatorLogFileName, i)))
		if err != nil {
			t.Fatal(err)
		}
		LogErrorOutput(t, validatorLogFile, "validator client", i)
	}

	t.Logf("Ending time: %s\n", time.Now().String())
}

// LogErrorOutput logs the output of a specific file.
func LogErrorOutput(t *testing.T, file io.Reader, title string, index int) {
	var errorLines []string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		currentLine := scanner.Text()
		if strings.Contains(currentLine, "level=error") {
			errorLines = append(errorLines, currentLine)
		}
	}
	if len(errorLines) < 1 {
		return
	}

	t.Logf("==================== Start of %s %d error output ==================\n", title, index)
	for _, err := range errorLines {
		t.Log(err)
	}
}

// WritePprofFiles writes the memory heap and cpu profile files to the test path.
func WritePprofFiles(testDir string, index int) error {
	url := fmt.Sprintf("http://127.0.0.1:%d/debug/pprof/heap", e2e.TestParams.Ports.PrysmBeaconNodePprofPort+index)
	filePath := filepath.Join(testDir, fmt.Sprintf(memoryHeapFileName, index))
	if err := writeURLRespAtPath(url, filePath); err != nil {
		return err
	}
	url = fmt.Sprintf("http://127.0.0.1:%d/debug/pprof/profile", e2e.TestParams.Ports.PrysmBeaconNodePprofPort+index)
	filePath = filepath.Join(testDir, fmt.Sprintf(cpuProfileFileName, index))
	return writeURLRespAtPath(url, filePath)
}

func writeURLRespAtPath(url, fp string) error {
	resp, err := http.Get(url) // #nosec G107 -- Safe, used internally
	if err != nil {
		return err
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			return
		}
	}()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	file, err := os.Create(filepath.Clean(fp))

	if err != nil {
		return err
	}
	if _, err = file.Write(body); err != nil {
		return err
	}
	return nil
}

// NewLocalConnection creates and returns GRPC connection on a given localhost port.
func NewLocalConnection(ctx context.Context, port int) (*grpc.ClientConn, error) {
	endpoint := fmt.Sprintf("127.0.0.1:%d", port)
	dialOpts := []grpc.DialOption{
		grpc.WithInsecure(),
	}
	conn, err := grpc.DialContext(ctx, endpoint, dialOpts...)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// NewLocalConnections returns number of GRPC connections, along with function to close all of them.
func NewLocalConnections(ctx context.Context, numConns int) ([]*grpc.ClientConn, func(), error) {
	conns := make([]*grpc.ClientConn, numConns)
	for i := 0; i < len(conns); i++ {
		conn, err := NewLocalConnection(ctx, e2e.TestParams.Ports.PrysmBeaconNodeRPCPort+i)
		if err != nil {
			return nil, nil, err
		}
		conns[i] = conn
	}
	return conns, func() {
		for _, conn := range conns {
			if err := conn.Close(); err != nil {
				log.Error(err)
			}
		}
	}, nil
}

// ComponentsStarted checks, sequentially, each provided component, blocks until all of the components are ready.
func ComponentsStarted(ctx context.Context, comps []e2etypes.ComponentRunner) error {
	for _, comp := range comps {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-comp.Started():
			continue
		}
	}
	return nil
}

// EpochTickerStartTime calculates the best time to start epoch ticker for a given genesis.
func EpochTickerStartTime(genesis *eth.Genesis) time.Time {
	epochSeconds := uint64(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().SecondsPerSlot))
	epochSecondsHalf := time.Duration(int64(epochSeconds*1000)/2) * time.Millisecond
	// Adding a half slot here to ensure the requests are in the middle of an epoch.
	middleOfEpoch := epochSecondsHalf + slots.DivideSlotBy(2 /* half a slot */)
	genesisTime := time.Unix(genesis.GenesisTime.Seconds, 0)
	// Offsetting the ticker from genesis so it ticks in the middle of an epoch, in order to keep results consistent.
	return genesisTime.Add(middleOfEpoch)
}

// WaitOnNodes waits on nodes to complete execution, accepts function that will be called when all nodes are ready.
func WaitOnNodes(ctx context.Context, nodes []e2etypes.ComponentRunner, nodesStarted func()) error {
	// Start nodes.
	g, ctx := errgroup.WithContext(ctx)
	for _, node := range nodes {
		node := node
		g.Go(func() error {
			return node.Start(ctx)
		})
	}

	// Mark set as ready (happens when all contained nodes report as started).
	go func() {
		for _, node := range nodes {
			select {
			case <-ctx.Done():
				return
			case <-node.Started():
				continue
			}
		}
		// When all nodes are done, signal the client. Client handles unresponsive components by setting up
		// a deadline for passed in context, and this ensures that nothing breaks if function below is never called.
		nodesStarted()
	}()

	return g.Wait()
}
