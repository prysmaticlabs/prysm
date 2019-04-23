package main

import (
	"context"
	"flag"
	"fmt"

	pb "github.com/prysmaticlabs/prysm/proto/cluster"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/keystore"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/grpc"
)

var (
	serverAddr  = flag.String("server", "", "The address of the gRPC server")
	podName     = flag.String("pod-name", "", "The name of the pod running this tool")
	keystoreDir = flag.String("keystore-dir", "", "The directory to generate keystore with received validator key")
	password    = flag.String("keystore-password", "", "The password to unlock the new keystore")
	numKeys     = flag.Uint64("keys", 1, "The number of keys to request")
)

func main() {
	flag.Parse()

	fmt.Printf("Starting client to fetch private key for pod %s\n", *podName)

	store := keystore.NewKeystore(*keystoreDir)

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

	for i, privateKey := range resp.PrivateKeys.PrivateKeys {
		pk, err := bls.SecretKeyFromBytes(privateKey)
		if err != nil {
			panic(err)
		}

		k := &keystore.Key{
			PublicKey: pk.PublicKey(),
			SecretKey: pk,
		}

		validatorKeyFile := *keystoreDir + params.BeaconConfig().ValidatorPrivkeyFileName + "-" + fmt.Sprintf("%d", i)
		if err := store.StoreKey(validatorKeyFile, k, *password); err != nil {
			panic(err)
		}

		fmt.Printf("New key written to %s\n", validatorKeyFile)
	}
}
