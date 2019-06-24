package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/PegasusMeteor/grpc-examples/grpc-tracing-intercepter"

	"github.com/PegasusMeteor/grpc-examples/grpc-consul/client/internel/consul"

	pb "github.com/PegasusMeteor/grpc-examples/proto/consul"

	"google.golang.org/grpc"
)

const (
	consulService = "consul://192.168.52.160:8500/helloworld" // consul中注册的服务地址
	defaultName   = "world"
	jaegerAgent   = "192.168.52.160:6831"
	serviceName   = "HelloClient"
)

func main() {
	consul.Init()

	tracer, closer, err := intercepter.NewJaegerTracer(serviceName, jaegerAgent)
	defer closer.Close()
	if err != nil {
		log.Printf("NewJaegerTracer err:", err.Error())
	}
	// Set up a connection to the server.
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	conn, err := grpc.DialContext(ctx, consulService, grpc.WithInsecure(), grpc.WithBalancerName("round_robin"), grpc.WithUnaryInterceptor(intercepter.ClientInterceptor(tracer)))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewGopherClient(conn)

	// Contact the server and print out its response.
	name := defaultName
	if len(os.Args) > 1 {
		name = os.Args[1]
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.SayHello(ctx, &pb.HelloRequest{Name: name})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Greeting: %s", r.Message)
}
