package snapshot

import (
	"strings"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	auth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	runtime "github.com/envoyproxy/go-control-plane/envoy/service/runtime/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"google.golang.org/protobuf/encoding/prototext"
)

func resourceType(res types.Resource) resource.Type {
	switch res.(type) {
	case *cluster.Cluster:
		return resource.ClusterType
	case *route.RouteConfiguration:
		return resource.RouteType
	case *route.ScopedRouteConfiguration:
		return resource.ScopedRouteType
	case *listener.Listener:
		return resource.ListenerType
	case *endpoint.ClusterLoadAssignment:
		return resource.EndpointType
	case *auth.Secret:
		return resource.SecretType
	case *runtime.Runtime:
		return resource.RuntimeType
	case *core.TypedExtensionConfig:
		return resource.ExtensionConfigType
	default:
		return ""
	}
}

func resourcesToMap(resources []types.Resource) map[string][]types.Resource {
	out := map[string][]types.Resource{}

	for _, res := range resources {
		t := resourceType(res)
		if _, ok := out[t]; !ok {
			out[t] = []types.Resource{res}
		} else {
			out[t] = append(out[t], res)
		}
	}

	return out
}

func DebugSnapshot(snapshot *cache.Snapshot) string {
	sb := strings.Builder{}

	for t, val := range snapshot.Resources {
		name, _ := cache.GetResponseTypeURL(types.ResponseType(t))
		sb.WriteString(name)
		sb.WriteString("\nVersion: ")
		sb.WriteString(val.Version)
		sb.WriteString("\n===============\n")
		for _, v := range val.Items {
			sb.WriteString(prototext.Format(v.Resource))
			sb.WriteString("----------\n")
		}

		sb.WriteString("\n\n")
	}

	return sb.String()
}
