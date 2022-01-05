package main

import (
	"context"
	"flag"
	"net"
	"os"
	"os/signal"
	"syscall"

	clusterservice "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	discoverygrpc "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	endpointservice "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	listenerservice "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	routeservice "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/wongnai/xds/debug"
	"github.com/wongnai/xds/snapshot"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	stopCtx, stop := context.WithCancel(context.Background())

	clientConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), nil).ClientConfig()
	if err != nil {
		klog.Fatal("Fail to create Kubernetes client config", err)
	}
	k8sClient, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		klog.Fatal("Fail to create Kubernetes client", err)
	}

	grpcServer := grpc.NewServer()
	snapshotter := snapshot.New(k8sClient)
	xdsServer := server.NewServer(stopCtx, snapshotter.MuxCache(), server.CallbackFuncs{
		StreamOpenFunc: func(ctx context.Context, streamID int64, typeURL string) error {
			klog.V(4).InfoS("StreamOpen", "streamID", streamID, "type", typeURL)
			return nil
		},
		StreamClosedFunc: func(streamID int64) {
			klog.V(4).InfoS("StreamClosed", "streamID", streamID)
		},
		DeltaStreamOpenFunc: func(ctx context.Context, streamID int64, typeURL string) error {
			klog.V(4).InfoS("DeltaStreamOpen", "streamID", streamID, "type", typeURL)
			return nil
		},
		DeltaStreamClosedFunc: func(streamID int64) {
			klog.V(4).InfoS("DeltaStreamClosed", "streamID", streamID)
		},
		StreamRequestFunc: func(streamID int64, request *discoverygrpc.DiscoveryRequest) error {
			klog.V(4).InfoS("StreamRequest", "streamID", streamID, "request", request)
			return nil
		},
		StreamResponseFunc: func(ctx context.Context, streamID int64, request *discoverygrpc.DiscoveryRequest, response *discoverygrpc.DiscoveryResponse) {
			klog.V(4).InfoS("StreamResponse", "streamID", streamID, "resourceNames", request.ResourceNames, "response", response)
		},
	})
	debugServer := debug.New(snapshotter.MuxCache())
	healthServer := health.NewServer()

	go func() {
		err := snapshotter.Start(stopCtx)
		if err != nil {
			klog.Fatal(err)
		}
	}()

	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	discoverygrpc.RegisterAggregatedDiscoveryServiceServer(grpcServer, xdsServer)
	endpointservice.RegisterEndpointDiscoveryServiceServer(grpcServer, xdsServer)
	clusterservice.RegisterClusterDiscoveryServiceServer(grpcServer, xdsServer)
	routeservice.RegisterRouteDiscoveryServiceServer(grpcServer, xdsServer)
	listenerservice.RegisterListenerDiscoveryServiceServer(grpcServer, xdsServer)

	lis, err := net.Listen("tcp", ":5000") //nolint:gosec
	if err != nil {
		klog.Fatal(err)
	}
	go func() {
		err = grpcServer.Serve(lis)
		if err != nil {
			klog.Fatal(err)
		}
	}()
	go debugServer.ListenAndServe()
	klog.Infoln("Server started")

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
	<-sigchan

	klog.Infoln("Stopping...")
	stop()
	healthServer.Shutdown()
	grpcServer.GracefulStop()
	lis.Close()
	klog.Infoln("Gracefully stopped")
}
