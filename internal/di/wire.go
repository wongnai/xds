//go:build wireinject

package di

import (
	"context"
	"github.com/google/wire"
	"github.com/wongnai/xds/debug"
	"google.golang.org/grpc"
)

type Servers struct {
	_GrpcHealth SideEffectGrpcHealthRegistered
	_Xds        XdsAllSideEffects
	_Reflection SideEffectGrpcReflectionRegistered
	_Channelz   SideEffectGrpcChannelzRegistered

	GrpcServer  *grpc.Server
	DebugServer *debug.Server
}

func InitializeServer(ctx context.Context, statsIntervalSeconds StatsIntervalSeconds) (Servers, func(), error) {
	wire.Build(
		KubernetesSet,
		GrpcSet,
		K8sXdsSet,
		XdsSet,
		wire.Struct(new(Servers), "*"),
	)
	return Servers{}, nil, nil
}
