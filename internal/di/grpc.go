package di

import (
	"os"

	"github.com/google/wire"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/channelz/service"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	reflectionservice "google.golang.org/grpc/reflection"
)

var GrpcSet = wire.NewSet(
	ProvideGrpcServer,
	ProvideGrpcHealthServer,
	ProvideSideEffectGrpcHealthRegistered,
	ProvideSideEffectGrpcReflectionRegisteredIfEnv,
	ProvideSideEffectGrpcChannelzRegistered,
)

func ProvideOtelGrpcServerOptions() []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	}
}

func ProvideGrpcServer(serverOptions []grpc.ServerOption) (*grpc.Server, func()) {
	server := grpc.NewServer(serverOptions...)
	return server, func() {
		server.GracefulStop()
	}
}

func ProvideGrpcHealthServer() (*health.Server, func()) {
	server := health.NewServer()
	return server, func() {
		server.Shutdown()
	}
}

type SideEffectGrpcHealthRegistered bool

// ProvideSideEffectGrpcHealthRegistered register a gRPC heath server implementation to the gRPC server
func ProvideSideEffectGrpcHealthRegistered(grpcServer *grpc.Server, healthServer *health.Server) SideEffectGrpcHealthRegistered {
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	return true
}

type SideEffectGrpcReflectionRegistered bool

// ProvideSideEffectGrpcReflectionRegistered register the gRPC reflection service to the gRPC server
func ProvideSideEffectGrpcReflectionRegistered(server *grpc.Server) SideEffectGrpcReflectionRegistered {
	reflectionservice.Register(server)
	return true
}

// ProvideSideEffectGrpcReflectionRegisteredIfEnv register the gRPC reflection service to the gRPC server
// but only if the envar ENABLE_GRPC_REFLECTION is set to exactly "true"
func ProvideSideEffectGrpcReflectionRegisteredIfEnv(server *grpc.Server) SideEffectGrpcReflectionRegistered {
	if os.Getenv("ENABLE_GRPC_REFLECTION") == "true" {
		return ProvideSideEffectGrpcReflectionRegistered(server)
	}
	return false
}

type SideEffectGrpcChannelzRegistered bool

// ProvideSideEffectGrpcChannelzRegistered register the gRPC channelz service to the gRPC server
func ProvideSideEffectGrpcChannelzRegistered(server *grpc.Server) SideEffectGrpcChannelzRegistered {
	service.RegisterChannelzServiceToServer(server)
	return true
}
