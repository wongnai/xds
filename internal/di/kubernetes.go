package di

import (
	"fmt"
	"net/http"

	"github.com/google/wire"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var KubernetesSet = wire.NewSet(
	ProvideClientConfig,
	ProvideK8sHTTPTransport,
	ProvideK8sHTTPClient,
	ProvideK8sClient,
)

func ProvideClientConfig() (*rest.Config, error) {
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), nil).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client config: %w", err)
	}
	return config, nil
}

type K8sHTTPTransport = http.RoundTripper

func ProvideK8sHTTPTransport(clientConfig *rest.Config) (K8sHTTPTransport, error) {
	transport, err := rest.TransportFor(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes HTTP transport: %w", err)
	}
	return transport, nil
}

type K8sHTTPClient = *http.Client

func ProvideK8sHTTPClient(transport K8sHTTPTransport, config *rest.Config) K8sHTTPClient {
	return &http.Client{
		Transport: otelhttp.NewTransport(transport),
		Timeout:   config.Timeout,
	}
}

func ProvideK8sClient(clientConfig *rest.Config, httpClient K8sHTTPClient) (*kubernetes.Clientset, error) {
	clientset, err := kubernetes.NewForConfigAndClient(clientConfig, httpClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}
	return clientset, nil
}
