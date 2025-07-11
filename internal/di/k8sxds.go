package di

import (
	"context"

	loadreportingservice "github.com/envoyproxy/go-control-plane/envoy/service/load_stats/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/google/wire"
	"github.com/wongnai/xds/debug"
	"github.com/wongnai/xds/meter"
	"github.com/wongnai/xds/report"
	"github.com/wongnai/xds/snapshot"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

var K8sXdsSet = wire.NewSet(
	ProvideSnapshotter,
	ProvideXdsServer,
	ProvideXdsLogger,
	ProvideDebugServer,
	ProvideLRSServer,
)

func ProvideSnapshotter(ctx context.Context, k8sClient kubernetes.Interface) (*snapshot.Snapshotter, func()) {
	stopCtx, stop := context.WithCancel(ctx)
	snapshotter := snapshot.New(k8sClient)

	go func() {
		err := snapshotter.Start(stopCtx)
		if err != nil {
			klog.Fatal(err)
		}
	}()

	return snapshotter, stop
}

func ProvideXdsServer(ctx context.Context, snapshotter *snapshot.Snapshotter, logger server.CallbackFuncs) (server.Server, func()) {
	stopCtx, stop := context.WithCancel(ctx)

	return server.NewServer(stopCtx, snapshotter.MuxCache(), logger), stop
}

func ProvideXdsLogger() server.CallbackFuncs {
	return meter.NewXdsServerCallbackFuncs()
}

// ProvideDebugServer create a debug server and immediately starts it
func ProvideDebugServer(snapshotter *snapshot.Snapshotter) *debug.Server {
	server := debug.New(snapshotter.MuxCache())

	go server.ListenAndServe()

	return server
}

type StatsIntervalSeconds = int64

func ProvideLRSServer(statsIntervalSeconds StatsIntervalSeconds) loadreportingservice.LoadReportingServiceServer {
	return report.NewServer(report.WithStatsIntervalInSeconds(statsIntervalSeconds))
}
