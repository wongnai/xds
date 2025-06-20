package di

import (
	clusterservice "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	discoverygrpc "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	endpointservice "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	listenerservice "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	loadreportingservice "github.com/envoyproxy/go-control-plane/envoy/service/load_stats/v3"
	routeservice "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/google/wire"
	"google.golang.org/grpc"
)

var XdsSet = wire.NewSet(
	ProvideSideEffectADSRegistered,
	ProvideSideEffectEDSRegistered,
	ProvideSideEffectCDSRegistered,
	ProvideSideEffectRDSRegistered,
	ProvideSideEffectLDSRegistered,
	ProvideSideEffectLRSRegistered,
	wire.Struct(new(XdsAllSideEffects), "*"),
)

type XdsAllSideEffects struct {
	_ADS SideEffectADSRegistered
	_EDS SideEffectEDSRegistered
	_CDS SideEffectCDSRegistered
	_RDS SideEffectRDSRegistered
	_LDS SideEffectLDSRegistered
	_LRS SideEffectLRSRegistered
}

type SideEffectADSRegistered bool

// ProvideSideEffectADSRegistered registers the Aggregated Discovery Service (ADS) with the gRPC server.
// ADS allows clients to receive all xDS resources (LDS, RDS, CDS, EDS, etc.) over a single gRPC stream.
func ProvideSideEffectADSRegistered(grpcServer *grpc.Server, xdsServer server.Server) SideEffectADSRegistered {
	discoverygrpc.RegisterAggregatedDiscoveryServiceServer(grpcServer, xdsServer)
	return true
}

type SideEffectEDSRegistered bool

// ProvideSideEffectEDSRegistered registers the Endpoint Discovery Service (EDS) with the gRPC server.
func ProvideSideEffectEDSRegistered(grpcServer *grpc.Server, xdsServer server.Server) SideEffectEDSRegistered {
	endpointservice.RegisterEndpointDiscoveryServiceServer(grpcServer, xdsServer)
	return true
}

type SideEffectCDSRegistered bool

// ProvideSideEffectCDSRegistered registers the Cluster Discovery Service (CDS) with the gRPC server.
func ProvideSideEffectCDSRegistered(grpcServer *grpc.Server, xdsServer server.Server) SideEffectCDSRegistered {
	clusterservice.RegisterClusterDiscoveryServiceServer(grpcServer, xdsServer)
	return true
}

type SideEffectRDSRegistered bool

// ProvideSideEffectRDSRegistered registers the Route Discovery Service (RDS) with the gRPC server.
func ProvideSideEffectRDSRegistered(grpcServer *grpc.Server, xdsServer server.Server) SideEffectRDSRegistered {
	routeservice.RegisterRouteDiscoveryServiceServer(grpcServer, xdsServer)
	return true
}

type SideEffectLDSRegistered bool

// ProvideSideEffectLDSRegistered registers the Listener Discovery Service (LDS) with the gRPC server.
func ProvideSideEffectLDSRegistered(grpcServer *grpc.Server, xdsServer server.Server) SideEffectLDSRegistered {
	listenerservice.RegisterListenerDiscoveryServiceServer(grpcServer, xdsServer)
	return true
}

type SideEffectLRSRegistered bool

// ProvideSideEffectLRSRegistered registers the Load Reporting Service (LRS) with the gRPC server.
func ProvideSideEffectLRSRegistered(grpcServer *grpc.Server, lrsServer loadreportingservice.LoadReportingServiceServer) SideEffectLRSRegistered {
	loadreportingservice.RegisterLoadReportingServiceServer(grpcServer, lrsServer)
	return true
}
