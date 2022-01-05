package snapshot

import (
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
)

// EmptyNodeID satisfies cachev3.NodeHash but always return empty string
type EmptyNodeID struct{}

func (e EmptyNodeID) ID(node *corev3.Node) string {
	return ""
}
