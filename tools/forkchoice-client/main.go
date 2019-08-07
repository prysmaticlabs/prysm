package main

import (
	"context"
	"encoding/hex"
	"flag"
	"reflect"
	"strconv"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	pb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var log = logrus.WithField("prefix", "forkchoice_client")

type endpoint []string

func (e *endpoint) String() string {
	return "gRPC endpoints"
}

func (e *endpoint) Set(value string) error {
	*e = append(*e, value)
	return nil
}

func main() {
	var endpts endpoint
	clients := make(map[string]pb.BeaconChainClient)

	flag.Var(&endpts, "endpoint", "Specify gRPC end points for beacon node")
	compare := flag.Bool("compare", false, "Enable head comparisons between all end points")
	flag.Parse()

	for i, endpt := range endpts {
		conn, err := grpc.Dial(endpt, grpc.WithInsecure())
		if err != nil {
			log.Fatalf("fail to dial: %v", err)
		}
		clients[endpt+strconv.Itoa(i)] = pb.NewBeaconChainClient(conn)
	}

	ticker := time.NewTicker(time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				if *compare {
					compareHeads(clients)
				} else {
					displayHeads(clients)
				}
			}
		}
	}()
	select {}
}

// log heads for all RPC end points
func displayHeads(clients map[string]pb.BeaconChainClient) {
	for endpt, client := range clients {
		head, err := client.GetChainHead(context.Background(), &ptypes.Empty{})
		if err != nil {
			log.Fatal(err)
		}
		logHead(endpt, head)
	}
}

// compare heads between all RPC end points, log the missmatch if there's one.
func compareHeads(clients map[string]pb.BeaconChainClient) {
	endpt1 := randomEndpt(clients)
	head1, err := clients[endpt1].GetChainHead(context.Background(), &ptypes.Empty{})
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("Compare all heads for head slot :%d", head1.BlockSlot)
	for endpt2, client := range clients {
		head2, err := client.GetChainHead(context.Background(), &ptypes.Empty{})
		if err != nil {
			log.Fatal(err)
		}
		if !reflect.DeepEqual(head1, head2) {
			log.Error("Uh oh! Head miss-matched!")
			logHead(endpt1, head1)
			logHead(endpt2, head2)
		}
	}
}

func logHead(endpt string, head *pb.ChainHead) {
	log.WithFields(
		logrus.Fields{
			"HeadSlot":      head.BlockSlot,
			"HeadRoot":      hex.EncodeToString(head.BlockRoot),
			"JustifiedSlot": head.JustifiedSlot,
			"JustifiedRoot": hex.EncodeToString(head.JustifiedBlockRoot),
			"FinalizedSlot": head.FinalizedSlot,
			"Finalizedroot": hex.EncodeToString(head.FinalizedBlockRoot),
		}).Info("Head from beacon node ", endpt)
}

func randomEndpt(clients map[string]pb.BeaconChainClient) string {
	for endpt := range clients {
		return endpt
	}
	return ""
}
