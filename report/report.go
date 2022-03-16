package report

import (
	"context"
	"sync"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	loadReportingService "github.com/envoyproxy/go-control-plane/envoy/service/load_stats/v3"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/wongnai/xds/meter"
	"go.opentelemetry.io/otel/metric"
	"k8s.io/klog/v2"
)

type server struct {
	loadReportingService.UnimplementedLoadReportingServiceServer

	lock           sync.Mutex
	nodesConnected map[string]bool

	statsIntervalInSeconds int64
	statsUpdateCounter     metric.Int64Counter
	nodeGauge              metric.Int64UpDownCounter
}

type Option func(s *server)

func NewServer(opts ...Option) loadReportingService.LoadReportingServiceServer {
	meter := metric.Must(meter.GetMeter())
	s := &server{
		nodesConnected:         make(map[string]bool),
		statsIntervalInSeconds: 300,
		statsUpdateCounter:     meter.NewInt64Counter("lrs_updates"),
		nodeGauge:              meter.NewInt64UpDownCounter("lrs_nodes"),
	}

	for _, o := range opts {
		o(s)
	}

	return s
}

func (s *server) StreamLoadStats(stream loadReportingService.LoadReportingService_StreamLoadStatsServer) error {
	var node *corev3.Node
	for {
		req, err := stream.Recv()
		if err != nil {
			if node != nil {
				s.removeNode(node)
			}
			return err
		}
		if node == nil {
			node = req.Node
		}

		s.HandleRequest(stream, req)
	}
}

func (s *server) HandleRequest(stream loadReportingService.LoadReportingService_StreamLoadStatsServer, request *loadReportingService.LoadStatsRequest) {
	nodeID := request.GetNode().GetId()

	s.statsUpdateCounter.Add(context.Background(), 1, meter.NodeIDAttrKey.String(nodeID))

	s.lock.Lock()
	defer s.lock.Unlock()

	if _, exist := s.nodesConnected[nodeID]; !exist {
		klog.V(4).InfoS("New node connected", "node_id", nodeID, "cluster_str", request.Node.Cluster)
		s.nodesConnected[nodeID] = true
		s.nodeGauge.Add(context.Background(), 1)

		err := stream.Send(&loadReportingService.LoadStatsResponse{
			Clusters:                  []string{"dummy_cluster"},
			LoadReportingInterval:     &duration.Duration{Seconds: s.statsIntervalInSeconds},
			ReportEndpointGranularity: true,
		})
		if err != nil {
			klog.Errorf("Unable to send response to node %s due to err: %s", nodeID, err)
			delete(s.nodesConnected, nodeID)
			klog.V(4).InfoS("Node disconnected", "node_id", nodeID, "cluster_str", request.Node.Cluster)
			s.nodeGauge.Add(context.Background(), -1)
		}
		return
	}

	for _, clusterStats := range request.ClusterStats {
		if len(clusterStats.UpstreamLocalityStats) > 0 {
			klog.V(4).InfoS("Got stats", "node_id", request.Node.Id, "cluster_str", request.Node.Cluster, "cluster_stats", clusterStats)
		}
	}
}

func (s *server) removeNode(node *corev3.Node) {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.nodesConnected, node.Id)

	klog.V(4).InfoS("Node disconnected", "node_id", node.Id, "cluster_str", node.Cluster)

	s.nodeGauge.Add(context.Background(), -1)
}

func WithStatsIntervalInSeconds(statsIntervalInSeconds int64) Option {
	return func(s *server) {
		s.statsIntervalInSeconds = statsIntervalInSeconds
	}
}
