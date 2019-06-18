package cmd

import (
	"context"
	"grpc-examples/grpc-consul/server/internel/consul"
	pb "grpc-examples/proto/consul"
	"net"

	"google.golang.org/grpc/health/grpc_health_v1"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	port = ":50051"
)

// server is used to implement helloworld.GreeterServer.
type server struct{}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Printf("Received: %v", in.Name)
	return &pb.HelloReply{Message: "Hello " + in.Name}, nil
}

// RegisterToConsul 调用RegisterService向consul中注册
func RegisterToConsul() {
	consul.RegisterService("127.0.0.1:8500", &consul.ConsulService{
		Name: "helloworld-gopher",
		Tag:  []string{"helloworld", "gopher"},
		IP:   "127.0.0.1",
		Port: 50051,
	})
}

//HealthImpl 定义一个空结构体用来进行健康检查
//HealthImpl 实现了HealthServer 这个接口
type HealthImpl struct{}

// Check 实现健康检查接口，这里直接返回健康状态，这里也可以有更复杂的健康检查策略，比如根据服务器负载来返回
func (h *HealthImpl) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	log.Println("health checking")
	return &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	}, nil
}
func (h *HealthImpl) Watch(req *grpc_health_v1.HealthCheckRequest, w grpc_health_v1.Health_WatchServer) error {
	return nil
}

func main() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterGopherServer(s, &server{})
	grpc_health_v1.RegisterHealthServer(s, &HealthImpl{})
	RegisterToConsul()
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
