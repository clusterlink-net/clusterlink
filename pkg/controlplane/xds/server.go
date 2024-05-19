// Copyright (c) The ClusterLink Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package xds

import (
	"context"

	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"google.golang.org/grpc"
)

// RegisterService registers an xDS service backed by Manager to the given gRPC server.
func RegisterService(ctx context.Context, manager *Manager, grpcServer *grpc.Server) {
	// create a combined mux cache of listeners, clusters and secrets
	muxCache := &cache.MuxCache{
		Classify: func(req *cache.Request) string {
			return req.TypeUrl
		},
		ClassifyDelta: func(req *cache.DeltaRequest) string {
			return req.TypeUrl
		},
		Caches: map[string]cache.Cache{
			resource.ClusterType:  manager.clusters,
			resource.ListenerType: manager.listeners,
		},
	}

	srv := server.NewServer(ctx, muxCache, nil)
	discovery.RegisterAggregatedDiscoveryServiceServer(grpcServer, srv)
}
