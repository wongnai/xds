package di

import (
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/google/wire"
	"google.golang.org/grpc"
)

type TestServer struct {
	DevServer

	Server server.Server
}

var TestSet = wire.NewSet(
	ProvideGrpcTestOption,
)

func ProvideGrpcTestOption() []grpc.ServerOption {
	return []grpc.ServerOption{}
}
