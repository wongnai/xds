package snapshot

import (
	"context"
	"time"

	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/log"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
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

	return &Snapshotter{
		ResyncPeriod: 10 * time.Minute,

		client:         client,
		servicesCache:  servicesCache,
		endpointsCache: endpointsCache,
		muxCache:       muxCache,
	}
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
