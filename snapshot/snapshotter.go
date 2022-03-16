package snapshot

import (
	"context"
	"sync"
	"time"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/log"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/wongnai/xds/meter"
	"go.opentelemetry.io/otel/metric"
	"golang.org/x/sync/errgroup"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

var Logger log.Logger = &log.LoggerFuncs{
	DebugFunc: func(s string, i ...interface{}) {
		klog.V(4).Infof(s, i...)
	},
	InfoFunc: func(s string, i ...interface{}) {
		klog.V(2).Infof(s, i...)
	},
	WarnFunc: func(s string, i ...interface{}) {
		klog.Warningf(s, i...)
	},
	ErrorFunc: func(s string, i ...interface{}) {
		klog.Errorf(s, i...)
	},
}

func mapTypeURL(typeURL string) string {
	switch typeURL {
	case resource.ListenerType, resource.RouteType, resource.ClusterType:
		return "services"
	case resource.EndpointType:
		return "endpoints"
	default:
		return ""
	}
}

type Snapshotter struct {
	ResyncPeriod time.Duration

	client         kubernetes.Interface
	servicesCache  cache.SnapshotCache
	endpointsCache cache.SnapshotCache
	muxCache       cache.MuxCache

	endpointResourceCache   map[string]endpointCacheItem
	resourcesByTypeLock     sync.RWMutex
	serviceResourcesByType  map[string][]types.Resource
	endpointResourcesByType map[string][]types.Resource
	apiGatewayStats         map[string]int
	kubeEventCounter        metric.Int64Counter
}

func New(client kubernetes.Interface) *Snapshotter {
	servicesCache := cache.NewSnapshotCache(false, EmptyNodeID{}, Logger)
	endpointsCache := cache.NewSnapshotCache(false, EmptyNodeID{}, Logger)
	muxCache := cache.MuxCache{
		Classify: func(r *cache.Request) string {
			return mapTypeURL(r.TypeUrl)
		},
		ClassifyDelta: func(r *cache.DeltaRequest) string {
			return mapTypeURL(r.TypeUrl)
		},
		Caches: map[string]cache.Cache{
			"services":  servicesCache,
			"endpoints": endpointsCache,
		},
	}

	ss := &Snapshotter{
		ResyncPeriod: 10 * time.Minute,

		client:         client,
		servicesCache:  servicesCache,
		endpointsCache: endpointsCache,
		muxCache:       muxCache,

		endpointResourceCache: map[string]endpointCacheItem{},
	}

	ss.kubeEventCounter = metric.Must(meter.GetMeter()).NewInt64Counter("xds_kube_events")
	_ = metric.Must(meter.GetMeter()).NewInt64GaugeObserver("xds_snapshot_resources", ss.snapshotResourceGaugeCallback)
	_ = metric.Must(meter.GetMeter()).NewInt64GaugeObserver("xds_apigateway_endpoints", ss.apiGatewayEndpointGaugeCallback)

	return ss
}

func (s *Snapshotter) MuxCache() *cache.MuxCache {
	return &s.muxCache
}

func (s *Snapshotter) Start(stopCtx context.Context) error {
	group, groupCtx := errgroup.WithContext(stopCtx)
	group.Go(func() error {
		return s.startServices(groupCtx)
	})
	group.Go(func() error {
		return s.startEndpoints(groupCtx)
	})
	return group.Wait()
}

func (s *Snapshotter) snapshotResourceGaugeCallback(_ context.Context, result metric.Int64ObserverResult) {
	for k, r := range s.getServiceResourcesByType() {
		result.Observe(int64(len(r)), meter.TypeURLAttrKey.String(k))
	}
	for k, r := range s.getEndpointResourcesByType() {
		result.Observe(int64(len(r)), meter.TypeURLAttrKey.String(k))
	}
}

func (s *Snapshotter) apiGatewayEndpointGaugeCallback(_ context.Context, result metric.Int64ObserverResult) {
	for k, stat := range s.getAPIGatewayStats() {
		result.Observe(int64(stat), meter.APIGatewayAttrKey.String(k))
	}
}

func (s *Snapshotter) setServiceResourcesByType(serviceResourcesByType map[string][]types.Resource) {
	s.resourcesByTypeLock.Lock()
	defer s.resourcesByTypeLock.Unlock()
	s.serviceResourcesByType = serviceResourcesByType
}

func (s *Snapshotter) getServiceResourcesByType() map[string][]types.Resource {
	s.resourcesByTypeLock.RLock()
	defer s.resourcesByTypeLock.RUnlock()
	return s.serviceResourcesByType
}

func (s *Snapshotter) setEndpointResourcesByType(endpointResourcesByType map[string][]types.Resource) {
	s.resourcesByTypeLock.Lock()
	defer s.resourcesByTypeLock.Unlock()
	s.endpointResourcesByType = endpointResourcesByType
}

func (s *Snapshotter) getEndpointResourcesByType() map[string][]types.Resource {
	s.resourcesByTypeLock.RLock()
	defer s.resourcesByTypeLock.RUnlock()
	return s.endpointResourcesByType
}

func (s *Snapshotter) setAPIGatewayStats(apiGatewayStats map[string]int) {
	s.resourcesByTypeLock.Lock()
	defer s.resourcesByTypeLock.Unlock()
	s.apiGatewayStats = apiGatewayStats
}

func (s *Snapshotter) getAPIGatewayStats() map[string]int {
	s.resourcesByTypeLock.RLock()
	defer s.resourcesByTypeLock.RUnlock()
	return s.apiGatewayStats
}
