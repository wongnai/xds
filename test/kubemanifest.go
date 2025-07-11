package test

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type K8SService struct {
	Name        string
	Namespace   string
	Ports       []corev1.ServicePort
	Annotations map[string]string
}

func (k *K8SService) AsK8S() *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Service"},
		ObjectMeta: metav1.ObjectMeta{
			Name:        k.Name,
			Namespace:   k.Namespace,
			Annotations: k.Annotations,
		},
		Spec: corev1.ServiceSpec{
			Ports: k.Ports,
		},
	}
}

type K8SEndpoint struct {
	Name      string
	Namespace string
	IP        []string
	Ports     []corev1.EndpointPort //nolint:staticcheck // We use Endpoint to simulate legacy Kube compatibility
}

func (k *K8SEndpoint) AsK8S() *corev1.Endpoints { //nolint:staticcheck // We use Endpoint to simulate legacy Kube compatibility
	addresses := make([]corev1.EndpointAddress, len(k.IP)) //nolint:staticcheck // See above
	for i, ip := range k.IP {
		addresses[i] = corev1.EndpointAddress{IP: ip} //nolint:staticcheck // See above
	}
	return &corev1.Endpoints{ //nolint:staticcheck // See above
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Endpoints"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k.Name,
			Namespace: k.Namespace,
		},
		Subsets: []corev1.EndpointSubset{{ //nolint:staticcheck // See above
			Addresses: addresses,
			Ports:     k.Ports,
		}},
	}
}
