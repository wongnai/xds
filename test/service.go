package test

import (
	"context"
	"net"
	"strconv"

	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type FakeService struct {
	mock.Mock

	listener   net.Listener
	grpcServer *grpc.Server
}

func NewFakeService(addr string) (*FakeService, error) {
	server := grpc.NewServer()
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	service := FakeService{
		listener:   listener,
		grpcServer: server,
	}

	grpc_health_v1.RegisterHealthServer(server, &service)

	go server.Serve(listener)

	return &service, nil
}

func (f *FakeService) Check(ctx context.Context, request *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	args := f.Called(ctx, request.Service)
	return args.Get(0).(*grpc_health_v1.HealthCheckResponse), args.Error(1)
}

func (f *FakeService) List(ctx context.Context, request *grpc_health_v1.HealthListRequest) (*grpc_health_v1.HealthListResponse, error) {
	panic("not implemented")
}

func (f *FakeService) Watch(request *grpc_health_v1.HealthCheckRequest, g grpc.ServerStreamingServer[grpc_health_v1.HealthCheckResponse]) error {
	panic("not implemented")
}

func (f *FakeService) Stop() {
	f.grpcServer.Stop()
	f.listener.Close()
}

func (f *FakeService) Addr() net.Addr {
	return f.listener.Addr()
}

func (f *FakeService) Host() string {
	host, _, err := net.SplitHostPort(f.Addr().String())
	if err != nil {
		panic(err)
	}
	return host
}

func (f *FakeService) Port() int32 {
	_, portStr, err := net.SplitHostPort(f.Addr().String())
	if err != nil {
		panic(err)
	}
	port, err := strconv.ParseInt(portStr, 10, 32)
	if err != nil {
		panic(err)
	}
	return int32(port)
}
