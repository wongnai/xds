package debug

import (
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	channelz "github.com/rantav/go-grpc-channelz"
	"k8s.io/apimachinery/pkg/util/json"
)

type Server struct {
	http.Server
	mux   *http.ServeMux
	cache *cache.MuxCache
}

func New(cache *cache.MuxCache) *Server {
	mux := http.NewServeMux()
	out := &Server{
		mux: mux,
		Server: http.Server{
			Addr:              ":9000",
			Handler:           mux,
			ReadTimeout:       60 * time.Second,
			ReadHeaderTimeout: 60 * time.Second,
			WriteTimeout:      60 * time.Minute, // for pprof
			IdleTimeout:       10 * time.Second,
		},
		cache: cache,
	}
	out.register()
	return out
}

func (s *Server) register() {
	s.mux.HandleFunc("/_hc", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	s.mux.Handle("/metrics", promhttp.Handler())

	s.mux.HandleFunc("/debug/pprof/", pprof.Index)
	s.mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	s.mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	s.mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	s.mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	s.mux.Handle("/channelz/", channelz.CreateHandler("/", ":5000"))

	s.mux.HandleFunc("/", s.snapshot)
}

func (s *Server) snapshot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "\t")

	out := map[string]cacheMarshaler{}
	for k, v := range s.cache.Caches {
		out[k] = cacheMarshaler{Cache: v}
	}

	encoder.Encode(out)
}
