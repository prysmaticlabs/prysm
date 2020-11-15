package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"

	"github.com/bazelbuild/buildtools/file"
	pb "github.com/prysmaticlabs/prysm/proto/cluster"
	_ "github.com/prysmaticlabs/prysm/shared/maxprocs"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/grpc"
)

var (
	serverAddr = flag.String("server", "", "The address of the gRPC server")
	podName    = flag.String("pod-name", "", "The name of the pod running this tool")
	numKeys    = flag.Uint64("keys", 1, "The number of keys to request")
	outputJSON = flag.String("output-json", "", "JSON file to write output to")
)

// UnencryptedKeysContainer defines the structure of the unecrypted key JSON file.
type UnencryptedKeysContainer struct {
	Keys []*UnencryptedKeys `json:"keys"`
}

// UnencryptedKeys is the inner struct of the JSON file.
type UnencryptedKeys struct {
	ValidatorKey  []byte `json:"validator_key"`
	WithdrawalKey []byte `json:"withdrawal_key"`
}

func main() {
	// Using Medalla as the default configuration.
	params.UseMedallaConfig()

	flag.Parse()

	fmt.Printf("Starting client to fetch private key for pod %s\n", *podName)

	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, *serverAddr, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	client := pb.NewPrivateKeyServiceClient(conn)

	resp, err := client.Request(ctx, &pb.PrivateKeyRequest{
		PodName:      *podName,
		NumberOfKeys: *numKeys,
	})
	if err != nil {
		panic(err)
	}

	keys := make([]*UnencryptedKeys, len(resp.PrivateKeys.PrivateKeys))

	for i, privateKey := range resp.PrivateKeys.PrivateKeys {
		keys[i] = &UnencryptedKeys{
			ValidatorKey:  privateKey,
			WithdrawalKey: privateKey,
		}
	}

	c := &UnencryptedKeysContainer{Keys: keys}
	enc, err := json.Marshal(c)
	if err != nil {
		panic(err)
	}
	if err := file.WriteFile(*outputJSON, enc); err != nil {
		panic(err)
	}

	fmt.Printf("Wrote %d keys\n", len(keys))
}
