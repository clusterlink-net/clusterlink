package grpc

import (
	"context"
	"crypto/tls"

	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"

	"github.com/clusterlink-org/clusterlink/pkg/controlplane"
	"github.com/clusterlink-org/clusterlink/pkg/util/grpc"
)

// Server implements an xDS server for dataplane dynamic configuration.
type Server struct {
	grpc.Server
}

// NewServer returns a new xDS server.
func NewServer(cp *controlplane.Instance, tlsConfig *tls.Config) *Server {
	// create a combined mux cache of listeners, clusters and secrets
	muxCache := &cache.MuxCache{
		Classify: func(req *cache.Request) string {
			return req.TypeUrl
		},
		ClassifyDelta: func(req *cache.DeltaRequest) string {
			return req.TypeUrl
		},
		Caches: map[string]cache.Cache{
			resource.ClusterType:  cp.GetXDSClusterManager(),
			resource.ListenerType: cp.GetXDSListenerManager(),
		},
	}

	srv := server.NewServer(context.Background(), muxCache, nil)
	s := &Server{
		Server: grpc.NewServer("controlplane-grpc", tlsConfig),
	}

	grpcServer := s.GetGRPCServer()
	discovery.RegisterAggregatedDiscoveryServiceServer(grpcServer, srv)

	return s
}
