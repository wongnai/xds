package test

import corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"

func FakeNode() *corev3.Node {
	return &corev3.Node{
		Id:      "fake",
		Cluster: "fake",
	}
}
