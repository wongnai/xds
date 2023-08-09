package snapshot

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel/metric"
	"net"
	"strconv"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	routerv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	managerv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/wongnai/xds/meter"
	"github.com/wongnai/xds/snapshot/apigateway"
	"google.golang.org/protobuf/types/known/anypb"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	k8scache "k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

func (s *Snapshotter) startServices(ctx context.Context) error {
	emit := func() {
		klog.Warning("emit before ready")
	}

	store := k8scache.NewUndeltaStore(func(v []interface{}) {
		emit()
	}, k8scache.DeletionHandlingMetaNamespaceKeyFunc)

	reflector := k8scache.NewReflector(&k8scache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return s.client.CoreV1().Services("").List(ctx, options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return s.client.CoreV1().Services("").Watch(ctx, options)
		},
	}, &corev1.Service{}, store, s.ResyncPeriod)

	var lastSnapshotHash uint64

	emit = func() {
		version := reflector.LastSyncResourceVersion()
		s.kubeEventCounter.Add(ctx, 1, metric.WithAttributes(meter.ResourceAttrKey.String("services")))

		services := sliceToService(store.List())
		resources := kubeServicesToResources(services)
		apiGatewayResources, apiGatewayStats := apigateway.FromKubeServices(services)
		merged := append(resources, apiGatewayResources...) //nolint:gocritic

		resourcesByType := resourcesToMap(merged)
		s.setServiceResourcesByType(resourcesByType)
		s.setAPIGatewayStats(apiGatewayStats)

		hash, err := resourcesHash(merged)
		if err == nil {
			if hash == lastSnapshotHash {
				klog.V(4).Info("new snapshot is equivalent to the previous one")
				return
			}
			lastSnapshotHash = hash
		} else {
			klog.Errorf("fail to hash snapshot: %s", err)
		}

		snapshot, err := cache.NewSnapshot(version, resourcesByType)
		if err != nil {
			panic(err)
		}

		s.servicesCache.SetSnapshot(ctx, "", snapshot)
	}

	reflector.Run(ctx.Done())
	return nil
}

func sliceToService(s []interface{}) []*corev1.Service {
	out := make([]*corev1.Service, len(s))
	for i, v := range s {
		out[i] = v.(*corev1.Service)
	}
	return out
}

// kubeServicesToResources convert list of Kubernetes services to
// - Listener for each ports
// - RouteConfiguration for those listeners
// - Cluster
func kubeServicesToResources(services []*corev1.Service) []types.Resource {
	var out []types.Resource

	router, _ := anypb.New(&routerv3.Router{})

	for _, svc := range services {
		fullName := fmt.Sprintf("%s.%s", svc.Name, svc.Namespace)
		for _, port := range svc.Spec.Ports {
			targetHostPort := net.JoinHostPort(fullName, port.Name)
			targetHostPortNumber := net.JoinHostPort(fullName, strconv.Itoa(int(port.Port)))
			routeConfig := &routev3.RouteConfiguration{
				Name: targetHostPortNumber,
				VirtualHosts: []*routev3.VirtualHost{
					{
						Name:    targetHostPort,
						Domains: []string{fullName, targetHostPort, targetHostPortNumber, svc.Name},
						Routes: []*routev3.Route{{
							Name: "default",
							Match: &routev3.RouteMatch{
								PathSpecifier: &routev3.RouteMatch_Prefix{},
							},
							Action: &routev3.Route_Route{
								Route: &routev3.RouteAction{
									ClusterSpecifier: &routev3.RouteAction_Cluster{
										Cluster: targetHostPort,
									},
								},
							},
						}},
					},
				},
			}

			manager, _ := anypb.New(&managerv3.HttpConnectionManager{
				HttpFilters: []*managerv3.HttpFilter{
					{
						Name: wellknown.Router,
						ConfigType: &managerv3.HttpFilter_TypedConfig{
							TypedConfig: router,
						},
					},
				},
				RouteSpecifier: &managerv3.HttpConnectionManager_RouteConfig{
					RouteConfig: routeConfig,
				},
			})

			svcListener := &listenerv3.Listener{
				Name: targetHostPortNumber,
				ApiListener: &listenerv3.ApiListener{
					ApiListener: manager,
				},
			}

			svcCluster := &clusterv3.Cluster{
				Name:                 targetHostPort,
				ClusterDiscoveryType: &clusterv3.Cluster_Type{Type: clusterv3.Cluster_EDS},
				LbPolicy:             clusterv3.Cluster_ROUND_ROBIN,
				EdsClusterConfig: &clusterv3.Cluster_EdsClusterConfig{
					EdsConfig: &corev3.ConfigSource{
						ConfigSourceSpecifier: &corev3.ConfigSource_Ads{
							Ads: &corev3.AggregatedConfigSource{},
						},
					},
				},
			}

			out = append(out, svcListener, routeConfig, svcCluster)
		}
	}

	return out
}
