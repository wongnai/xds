package snapshot

import (
	"fmt"
	"math/rand"
	"testing"

	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var resourceHashTestSet1 = []types.Resource{
	&endpointv3.ClusterLoadAssignment{
		ClusterName: "test1",
	},
	&endpointv3.ClusterStats{
		ClusterName: "test1",
	},
}

var resourceHashTestSet2 = []types.Resource{
	&endpointv3.ClusterLoadAssignment{
		ClusterName: "test2",
	},
	&endpointv3.ClusterStats{
		ClusterName: "test2",
	},
}

func TestResourcesHash(t *testing.T) {
	a, err := resourcesHash(resourceHashTestSet1)
	require.NoError(t, err)

	b, err := resourcesHash(resourceHashTestSet2)
	require.NoError(t, err)

	assert.NotEqual(t, a, b)
}

func BenchmarkResourcesHash(b *testing.B) {
	resources := make([]types.Resource, 0, 100)
	for i := 0; i < 100; i++ {
		resources = append(resources, &endpointv3.ClusterLoadAssignment{
			ClusterName: fmt.Sprintf("test%d", i),
			Endpoints: []*endpointv3.LocalityLbEndpoints{
				{
					Priority: rand.Uint32(), //nolint:gosec // test can use non-csprng
				},
			},
		})
	}

	for b.Loop() {
		_, err := resourcesHash(resources)
		if err != nil {
			b.Fatal(err)
		}
	}
}
