package test_test

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/wongnai/xds/internal/di"
	"github.com/wongnai/xds/snapshot/apigateway"
	"github.com/wongnai/xds/test"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/xds"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/klog/v2"
)

// xdsServerBind is where the xDS Server is listening. Since the port is :0, use s.listener.Addr().String() to get the actual address
const xdsServerBind = "127.2.0.1:0"

type XdsIntegrationTestSuite struct {
	suite.Suite
	di.TestServer

	listener           net.Listener
	kube               *fake.Clientset
	activeFakeServices []*test.FakeService

	fakeServiceIP uint8
}

func (s *XdsIntegrationTestSuite) SetupSuite() {
	listener, err := net.Listen("tcp", xdsServerBind)
	s.Require().NoError(err)
	s.listener = listener

	go func() {
		err := s.TestServer.GrpcServer.Serve(listener)
		if err != nil {
			s.T().Error(err)
		}
	}()
}

func (s *XdsIntegrationTestSuite) TearDownTest() {
	// TODO: Clear s.kube.Tracker
	s.kube.ClearActions()
	for _, service := range s.activeFakeServices {
		service.AssertExpectations(s.T())
		service.Stop()
	}
	s.activeFakeServices = nil
	s.fakeServiceIP = 0
	klog.Flush()
}

func (s *XdsIntegrationTestSuite) TearDownSuite() {
	s.TestServer.GrpcServer.Stop()
	s.listener.Close()
}

func (s *XdsIntegrationTestSuite) getClient(target string) grpc_health_v1.HealthClient {
	xdsBuilder, err := xds.NewXDSResolverWithConfigForTesting([]byte(fmt.Sprintf(`{
		"xds_servers": [{
			"server_uri": "%s",
			"channel_creds": [{"type": "insecure"}],
            "server_features": ["xds_v3"]
		}],
		"node": {
			"id": "test",
			"locality": {
				"zone" : "test"
			}
		}
	}`, s.listener.Addr().String())))
	s.Require().NoError(err)
	client, err := grpc.NewClient(target, grpc.WithResolvers(xdsBuilder), grpc.WithTransportCredentials(insecure.NewCredentials()))
	s.Require().NoError(err)

	healthClient := grpc_health_v1.NewHealthClient(client)
	return healthClient
}

func (s *XdsIntegrationTestSuite) createFakeService(serviceName string, namespace string, port int32, register bool) *test.FakeService {
	svc, err := test.NewFakeService(fmt.Sprintf("%s:%d", s.getFakeServiceIP(), port))
	s.Require().NoError(err)
	svc.Test(s.T())
	s.activeFakeServices = append(s.activeFakeServices, svc)

	if register {
		s.createKubeService(serviceName, namespace, port)
		s.createKubeEndpoint(serviceName, namespace, svc.Host(), svc.Port())
	}

	return svc
}

func (s *XdsIntegrationTestSuite) createKubeService(serviceName string, namespace string, servicePort int32) {
	svc := &test.K8SService{
		Name:      serviceName,
		Namespace: namespace,
		Ports: []corev1.ServicePort{{
			Name:     "grpc",
			Port:     servicePort,
			Protocol: corev1.ProtocolTCP,
		}},
	}
	err := s.kube.Tracker().Add(svc.AsK8S())
	s.Require().NoError(err)
}

func (s *XdsIntegrationTestSuite) createKubeEndpoint(serviceName string, namespace string, ip string, port int32) {
	endpoint := &test.K8SEndpoint{
		Name:      serviceName,
		Namespace: namespace,
		IP:        []string{ip},
		Ports: []corev1.EndpointPort{{
			Name: "grpc",
			Port: port,
		}},
	}
	err := s.kube.Tracker().Add(endpoint.AsK8S()) //nolint:staticcheck // We use Endpoint to simulate legacy Kube compatibility
	s.Require().NoError(err)
}

func (s *XdsIntegrationTestSuite) getFakeServiceIP() string {
	out := s.fakeServiceIP
	s.fakeServiceIP += 1

	return fmt.Sprintf("127.2.1.%d", out)
}

func (s *XdsIntegrationTestSuite) TestValidTarget() {
	svc := s.createFakeService("app", "default", 0, false)
	s.createKubeService("app", "default", 1)
	s.createKubeEndpoint("app", "default", svc.Host(), svc.Port())

	// Test that the client is able to connect
	client := s.getClient("xds:///app.default:1")
	s.T().Run("initial", func(t *testing.T) {
		svc.On("Check", mock.Anything, "test").Return(&grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil)

		_, err := client.Check(t.Context(), &grpc_health_v1.HealthCheckRequest{Service: "test"})
		assert.NoError(t, err)
	})

	// Test that once the backend IP change, the client connects to the new one
	svc.Stop() // Stop the old one

	// Create unrelated apps to simulate unrelated events
	s.createKubeService("app", "unused", 1)
	s.createKubeEndpoint("app", "unused", "0.0.0.1", 1)

	svc = s.createFakeService("app", "default", 0, false)
	err := s.kube.Tracker().Update(
		schema.GroupVersionResource{Group: "", Version: "v1", Resource: "endpoints"},
		&corev1.Endpoints{ //nolint:staticcheck // We use Endpoint to simulate legacy Kube compatibility
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Endpoints",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "app",
				Namespace: "default",
			},
			Subsets: []corev1.EndpointSubset{{ //nolint:staticcheck // See above
				Addresses: []corev1.EndpointAddress{{ //nolint:staticcheck // See above
					IP: svc.Host(),
				}},
				Ports: []corev1.EndpointPort{ //nolint:staticcheck // See above
					{
						Name: "grpc",
						Port: svc.Port(),
					},
					{
						Name: "http",
						Port: 9999,
					},
				},
			}},
		},
		"default",
	)
	s.Require().NoError(err)

	// It doesn't seems that XDS propagation works in test??
	// s.T().Run("updated", func(t *testing.T) {
	//	svc.On("Check", mock.Anything, "test2").Return(&grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil)
	//
	//	_, err := client.Check(t.Context(), &grpc_health_v1.HealthCheckRequest{Service: "test2"})
	//	assert.NoError(t, err)
	// })
}

func (s *XdsIntegrationTestSuite) TestApiGateway() {
	svc1 := s.createFakeService("apigwbackend1", "default", 50000, false)
	svc1Manifest := &test.K8SService{
		Name:      "apigwbackend1",
		Namespace: "default",
		Annotations: map[string]string{
			apigateway.NameAnnotation:    "apigw1,apigw2",
			apigateway.ServiceAnnotation: "grpc.health.v1.Health,lmwn.inexists.v1.Test",
		},
		Ports: []corev1.ServicePort{{
			Name:     "grpc",
			Port:     50000,
			Protocol: corev1.ProtocolTCP,
		}},
	}
	err := s.kube.Tracker().Add(svc1Manifest.AsK8S())
	s.Require().NoError(err)
	s.createKubeEndpoint("apigwbackend1", "default", svc1.Host(), svc1.Port())

	svc2 := s.createFakeService("apigwbackend2", "default", 50001, false)
	svc2Manifest := &test.K8SService{
		Name:      "apigwbackend2",
		Namespace: "default",
		Annotations: map[string]string{
			apigateway.NameAnnotation:    "apigw1,apigw2",
			apigateway.ServiceAnnotation: "",
		},
		Ports: []corev1.ServicePort{{
			Name:     "grpc",
			Port:     50001,
			Protocol: corev1.ProtocolTCP,
		}},
	}
	err = s.kube.Tracker().Add(svc2Manifest.AsK8S())
	s.Require().NoError(err)
	s.createKubeEndpoint("apigwbackend2", "default", svc2.Host(), svc2.Port())

	svc1.On("Check", mock.Anything, "test").Return(&grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil)

	client := s.getClient("xds:///apigw1")
	_, err = client.Check(s.T().Context(), &grpc_health_v1.HealthCheckRequest{Service: "test"})
	require.NoError(s.T(), err)
}

func TestXdsIntegration(t *testing.T) {
	kube := fake.NewClientset()

	testServer, stop, err := di.InitializeTestServer(t.Context(), kube, 1)
	require.NoError(t, err)
	defer stop()

	suite.Run(t, &XdsIntegrationTestSuite{
		TestServer: testServer,
		kube:       kube,
	})
}
