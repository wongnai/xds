package snapshot

import (
	"context"
	"fmt"
	"sort"

	"github.com/ccoveille/go-safecast"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/wongnai/xds/meter"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/protobuf/types/known/wrapperspb"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	k8scache "k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

type endpointCacheItem struct {
	version   string
	resources []types.Resource
}

func (s *Snapshotter) startEndpoints(ctx context.Context) error {
	emit := func() {}

	store := k8scache.NewUndeltaStore(func(v []interface{}) {
		emit()
	}, k8scache.DeletionHandlingMetaNamespaceKeyFunc)

	reflector := k8scache.NewReflector(&k8scache.ListWatch{
		ListWithContextFunc: func(ctx context.Context, options metav1.ListOptions) (runtime.Object, error) {
			return s.client.CoreV1().Endpoints("").List(ctx, options)
		},
		WatchFuncWithContext: func(ctx context.Context, options metav1.ListOptions) (watch.Interface, error) {
			return s.client.CoreV1().Endpoints("").Watch(ctx, options)
		},
	}, &corev1.Endpoints{}, store, s.ResyncPeriod) //nolint:staticcheck // We use deprecated API to support legacy Kubernetes

	var lastSnapshotHash uint64

	emit = func() {
		version := reflector.LastSyncResourceVersion()
		s.kubeEventCounter.Add(ctx, 1, metric.WithAttributes(meter.ResourceAttrKey.String("endpoints")))

		endpoints := sliceToEndpoints(store.List())
		endpointsResources := s.kubeEndpointsToResources(endpoints)
		hash, err := resourcesHash(endpointsResources)
		if err == nil {
			if hash == lastSnapshotHash {
				klog.V(5).Info("new snapshot is equivalent to the previous one")
				return
			}
			lastSnapshotHash = hash
		} else {
			klog.Errorf("fail to hash snapshot: %s", err)
		}

		resourcesByType := resourcesToMap(endpointsResources)
		s.setEndpointResourcesByType(resourcesByType)

		snapshot, err := cache.NewSnapshot(version, resourcesByType)
		if err != nil {
			panic(err)
		}

		s.endpointsCache.SetSnapshot(ctx, "", snapshot)
	}

	reflector.Run(ctx.Done())
	return nil
}

func sliceToEndpoints(s []interface{}) []*corev1.Endpoints { //nolint:staticcheck // We use deprecated API to support legacy Kubernetes
	out := make([]*corev1.Endpoints, len(s)) //nolint:staticcheck
	for i, v := range s {
		out[i] = v.(*corev1.Endpoints) //nolint:staticcheck
	}
	return out
}

// kubeServicesToResources convert list of Kubernetes endpoints to Endpoint
func (s *Snapshotter) kubeEndpointsToResources(endpoints []*corev1.Endpoints) []types.Resource { //nolint:staticcheck // We use deprecated API to support legacy Kubernetes
	var out []types.Resource

	for _, ep := range endpoints {
		out = append(out, s.kubeEndpointToResources(ep)...)
	}

	return out
}

func (s *Snapshotter) kubeEndpointToResources(ep *corev1.Endpoints) []types.Resource { //nolint:staticcheck // We use deprecated API to support legacy Kubernetes
	name, err := k8scache.MetaNamespaceKeyFunc(ep)
	if err != nil {
		klog.Errorf("fail to get object key: %s", err)
		return nil
	}
	if val, ok := s.endpointResourceCache[name]; ok && val.version == ep.ResourceVersion {
		return val.resources
	}

	var out []types.Resource

	for _, subset := range ep.Subsets {
		for _, port := range subset.Ports {
			var portName string
			if port.Name == "" {
				portName = fmt.Sprintf("%s.%s:%d", ep.Name, ep.Namespace, port.Port)
			} else {
				portName = fmt.Sprintf("%s.%s:%s", ep.Name, ep.Namespace, port.Name)
			}

			cla := &endpointv3.ClusterLoadAssignment{
				ClusterName: portName,
				Endpoints: []*endpointv3.LocalityLbEndpoints{
					{
						LoadBalancingWeight: wrapperspb.UInt32(1),
						Locality:            &corev3.Locality{},
						LbEndpoints:         []*endpointv3.LbEndpoint{},
					},
				},
			}
			out = append(out, cla)

			sortedAddresses := subset.Addresses
			sort.SliceStable(sortedAddresses, func(i, j int) bool {
				l := sortedAddresses[i].IP
				r := sortedAddresses[j].IP
				return l < r
			})

			for _, addr := range sortedAddresses {
				hostname := addr.Hostname
				if hostname == "" && addr.TargetRef != nil {
					hostname = fmt.Sprintf("%s.%s", addr.TargetRef.Name, addr.TargetRef.Namespace)
				}
				if hostname == "" && addr.NodeName != nil {
					hostname = *addr.NodeName
				}
				portU32, err := safecast.ToUint32(port.Port)
				if err != nil {
					panic(err)
				}

				cla.Endpoints[0].LbEndpoints = append(cla.Endpoints[0].LbEndpoints, &endpointv3.LbEndpoint{
					HostIdentifier: &endpointv3.LbEndpoint_Endpoint{
						Endpoint: &endpointv3.Endpoint{
							Address: &corev3.Address{
								Address: &corev3.Address_SocketAddress{
									SocketAddress: &corev3.SocketAddress{
										Protocol: corev3.SocketAddress_TCP,
										Address:  addr.IP,
										PortSpecifier: &corev3.SocketAddress_PortValue{
											PortValue: portU32,
										},
									},
								},
							},
							Hostname: hostname,
						},
					},
				})
			}
		}
	}

	s.endpointResourceCache[name] = endpointCacheItem{
		version:   ep.ResourceVersion,
		resources: out,
	}

	return out
}
