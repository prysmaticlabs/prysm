package main

import (
	"flag"
	"fmt"
	"net"

	recaptcha "github.com/prestonvanloon/go-recaptcha"
	faucetpb "github.com/prysmaticlabs/prysm/tools/faucet/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	port            = flag.Int("port", 8000, "Port to server gRPC service")
	recaptchaSecret = flag.String("recaptcha_secret", "", "Secret to verify recaptcha")
)

func main() {
	flag.Parse()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		panic(err)
	}
	s := grpc.NewServer()
	fmt.Println("recaptcha = " + *recaptchaSecret)
	faucetpb.RegisterFaucetServiceServer(s, &faucetServer{
		r: recaptcha.Recaptcha{RecaptchaPrivateKey: *recaptchaSecret},
	})

	reflection.Register(s)

	fmt.Printf("Serving gRPC requests on port %d\n", *port)
	s.Serve(lis)
}
