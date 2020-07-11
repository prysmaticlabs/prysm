package main

import (
	"flag"
	"fmt"
	"net"

	"github.com/prestonvanloon/go-recaptcha"
	faucetpb "github.com/prysmaticlabs/prysm/proto/faucet"
	_ "github.com/prysmaticlabs/prysm/shared/maxprocs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	port            = flag.Int("port", 8000, "Port to server gRPC service")
	recaptchaSecret = flag.String("recaptcha_secret", "", "Secret to verify recaptcha")
	rpcPath         = flag.String("rpc", "", "RPC address of a running geth node")
	privateKey      = flag.String("private-key", "", "The private key of funder")
	minScore        = flag.Float64("min-score", 0.9, "Minimum captcha score.")
)

func main() {
	flag.Parse()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		panic(err)
	}
	s := grpc.NewServer()
	fmt.Println("recaptcha = " + *recaptchaSecret)
	faucetpb.RegisterFaucetServiceServer(s,
		newFaucetServer(
			recaptcha.Recaptcha{RecaptchaPrivateKey: *recaptchaSecret},
			*rpcPath,
			*privateKey,
			*minScore,
		),
	)

	reflection.Register(s)
	go counterWatcher()

	fmt.Printf("Serving gRPC requests on port %d\n", *port)
	if err := s.Serve(lis); err != nil {
		fmt.Printf("Error: %v", err)
	}
}
