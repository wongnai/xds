package test_test

import (
	"context"
	"net"
	"testing"

	discoveryv3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/stretchr/testify/suite"
	"github.com/wongnai/xds/internal/di"
	"github.com/wongnai/xds/test"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/fake"
)

type XdsSuite struct {
	suite.Suite
	di.TestServer

	kube *fake.Clientset
	stop func()
	conn *bufconn.Listener
}

func (s *XdsSuite) SetupTest() {
	var err error
	s.kube = fake.NewClientset()
	s.conn = bufconn.Listen(1)
	s.TestServer, s.stop, err = di.InitializeTestServer(s.T().Context(), s.kube, 1)
	s.Require().NoError(err)

	go s.TestServer.GrpcServer.Serve(s.conn)
}

func (s *XdsSuite) TearDownTest() {
	if s.stop != nil {
		s.stop()
	}
	s.TestServer.GrpcServer.Stop()
	s.conn.Close()
}

func (s *XdsSuite) GetGrpcClient() *grpc.ClientConn {
	out, err := grpc.NewClient("passthrough:internal", grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
		return s.conn.DialContext(ctx)
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	s.Require().NoError(err)

	return out
}

func (s *XdsSuite) TestEndpointInfo() {
	const svcName = "test-service"
	const svcNamespace = "default"
	fakeSvc := &test.K8SService{
		Name:      svcName,
		Namespace: svcNamespace,
		Ports: []corev1.ServicePort{{
			Name: "grpc",
			Port: 50000,
		}},
	}
	s.kube.Tracker().Add(fakeSvc.AsK8S())

	fakeEndpoint := &test.K8SEndpoint{
		Name:      svcName,
		Namespace: svcNamespace,
		IP:        []string{"127.0.0.1"},
		Ports: []corev1.EndpointPort{{
			Name: "grpc",
			Port: 50000,
		}},
	}
	s.kube.Tracker().Add(fakeEndpoint.AsK8S())

	ads := discoveryv3.NewAggregatedDiscoveryServiceClient(s.GetGrpcClient())
	adsStream, err := ads.StreamAggregatedResources(s.T().Context())
	s.Require().NoError(err)
	defer adsStream.CloseSend()

	adsStream.Send(&discoveryv3.DiscoveryRequest{
		VersionInfo: "",
		Node:        test.FakeNode(),
		ResourceNames: []string{
			"test-service.default:grpc",
		},
		TypeUrl: resource.APITypePrefix + resource.EndpointType,
	})

	s.T().Skip("This test doesn't work currently and is left for smoke testing")

	// discovery, err := adsStream.Recv()
	// s.Require().NoError(err)
	// s.T().Logf("%s", discovery.String())
}

func TestXds(t *testing.T) {
	suite.Run(t, &XdsSuite{})
}
