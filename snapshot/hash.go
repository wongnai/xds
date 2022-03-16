package snapshot

import (
	"sort"

	"github.com/cespare/xxhash"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"google.golang.org/protobuf/proto"
)

// resourcesHash hash the provided resources
// This function may mutate input to make it deterministic
func resourcesHash(resources []types.Resource) (uint64, error) {
	hasher := xxhash.New()

	sort.SliceStable(resources, func(i, j int) bool {
		nameI := cache.GetResourceName(resources[i])
		nameJ := cache.GetResourceName(resources[j])

		return nameI < nameJ
	})

	var buf []byte
	var err error

	for _, resource := range resources {
		buf, err = proto.MarshalOptions{
			Deterministic: true,
		}.MarshalAppend(buf, resource)
		if err != nil {
			return 0, err
		}
		hasher.Write(buf)
		hasher.Write([]byte{0})
		buf = buf[:0]
	}

	return hasher.Sum64(), nil
}
