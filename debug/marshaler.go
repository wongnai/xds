package debug

import (
	"encoding/json"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"google.golang.org/protobuf/encoding/protojson"
)

type cacheMarshaler struct {
	cache.Cache
}

func (c cacheMarshaler) MarshalJSON() ([]byte, error) {
	snapshotCache, ok := c.Cache.(cache.SnapshotCache)
	if !ok {
		return nil, nil
	}

	out := map[string]map[resource.Type]interface{}{}
	nodes := []string{""} // TODO: how do i not hardcode this...

	for _, node := range nodes {
		snapshot, err := snapshotCache.GetSnapshot(node)
		if err != nil {
			return nil, err
		}
		nodeMap := map[resource.Type]interface{}{}
		out[node] = nodeMap

		for i := types.ResponseType(0); i < types.UnknownType; i++ {
			if len(snapshot.Resources[i].Items) == 0 {
				continue
			}
			typeName, _ := cache.GetResponseTypeURL(i)
			nodeMap[typeName] = resourcesMarshaler{snapshot.Resources[i]}
		}
	}

	return json.Marshal(out)
}

type resourcesMarshaler struct {
	cache.Resources
}

func (r resourcesMarshaler) MarshalJSON() ([]byte, error) {
	type outMap struct {
		Version string                       `json:"version"`
		Items   map[string]resourceMarshaler `json:"items"`
	}

	out := outMap{
		Version: r.Version,
		Items:   map[string]resourceMarshaler{},
	}

	for k, v := range r.Resources.Items {
		out.Items[k] = resourceMarshaler{Resource: v.Resource}
	}

	return json.Marshal(out)
}

type resourceMarshaler struct {
	types.Resource
}

func (r resourceMarshaler) MarshalJSON() ([]byte, error) {
	return protojson.Marshal(r.Resource)
}
