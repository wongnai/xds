package meter

import (
	"context"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discoverygrpc "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"k8s.io/klog/v2"
)

var (
	TypeURLAttrKey    attribute.Key = "type_url"
	APIGatewayAttrKey attribute.Key = "api_gateway"
	ResourceAttrKey   attribute.Key = "resource"
)

func NewXdsServerCallbackFuncs() server.CallbackFuncs {
	meter := GetMeter()
	streamGauge, _ := meter.Int64UpDownCounter("xds_server_streams")
	deltaStreamGauge, _ := meter.Int64UpDownCounter("xds_server_delta_streams")
	requestCounter, _ := meter.Int64Counter("xds_server_stream_requests")
	responseCounter, _ := meter.Int64Counter("xds_server_stream_responses")

	return server.CallbackFuncs{
		StreamOpenFunc: func(ctx context.Context, streamID int64, typeURL string) error {
			streamGauge.Add(ctx, 1)
			klog.V(4).InfoS("StreamOpen", "streamID", streamID, "type", typeURL)
			return nil
		},
		StreamClosedFunc: func(streamID int64, node *corev3.Node) {
			streamGauge.Add(context.Background(), -1)
			klog.V(4).InfoS("StreamClosed", "streamID", streamID)
		},
		DeltaStreamOpenFunc: func(ctx context.Context, streamID int64, typeURL string) error {
			deltaStreamGauge.Add(ctx, 1)
			klog.V(4).InfoS("DeltaStreamOpen", "streamID", streamID, "type", typeURL)
			return nil
		},
		DeltaStreamClosedFunc: func(streamID int64, node *corev3.Node) {
			deltaStreamGauge.Add(context.Background(), -1)
			klog.V(4).InfoS("DeltaStreamClosed", "streamID", streamID)
		},
		StreamRequestFunc: func(streamID int64, request *discoverygrpc.DiscoveryRequest) error {
			requestCounter.Add(context.Background(), 1, metric.WithAttributes(TypeURLAttrKey.String(request.GetTypeUrl())))
			klog.V(4).InfoS("StreamRequest", "streamID", streamID, "request", request)
			return nil
		},
		StreamResponseFunc: func(ctx context.Context, streamID int64, request *discoverygrpc.DiscoveryRequest, response *discoverygrpc.DiscoveryResponse) {
			responseCounter.Add(ctx, 1, metric.WithAttributes(TypeURLAttrKey.String(request.GetTypeUrl())))
			klog.V(4).InfoS("StreamResponse", "streamID", streamID, "resourceNames", request.ResourceNames, "response", response)
		},
	}
}
