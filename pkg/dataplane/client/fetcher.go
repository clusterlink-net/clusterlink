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

package client

import (
	"context"
	"fmt"
	"strings"
	"sync"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	client "github.com/envoyproxy/go-control-plane/pkg/client/sotw/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/clusterlink-net/clusterlink/pkg/controlplane/api"
	"github.com/clusterlink-net/clusterlink/pkg/dataplane/server"
)

type fetcher struct {
	client       client.ADSClient
	resourceType string
	dataplane    *server.Dataplane
	logger       *logrus.Entry
	clusterLock  sync.Mutex
	listenerLock sync.Mutex
}

func (f *fetcher) handleClusters(resources []*anypb.Any) error {
	clusters := make(map[string]bool)

	f.clusterLock.Lock()
	defer f.clusterLock.Unlock()
	for _, r := range resources {
		c := &cluster.Cluster{}
		err := anypb.UnmarshalTo(r, c, proto.UnmarshalOptions{})
		if err != nil {
			return err
		}

		f.logger.Debugf("Cluster: %s.", c.Name)
		f.dataplane.AddCluster(c)
		clusters[c.Name] = true
	}
	// Delete existing clusters if its not present in the reources fetched
	for cn := range f.dataplane.GetClusters() {
		if _, ok := clusters[cn]; ok {
			// Cluster exists in the resources fetched
			continue
		}
		f.logger.Debugf("Remove Cluster: %s.", cn)
		f.dataplane.RemoveCluster(cn)
	}
	return nil
}

func (f *fetcher) handleListeners(resources []*anypb.Any) error {
	listeners := make(map[string]bool)

	f.listenerLock.Lock()
	defer f.listenerLock.Unlock()
	// Add any new listeners created
	for _, r := range resources {
		l := &listener.Listener{}
		err := anypb.UnmarshalTo(r, l, proto.UnmarshalOptions{})
		if err != nil {
			return err
		}
		f.logger.Debugf("Listener: %s.", l.Name)
		f.dataplane.AddListener(l)
		listeners[strings.TrimPrefix(l.Name, api.ImportListenerPrefix)] = true
	}
	// Delete existing listeners if its not present in the reources fetched
	for ln := range f.dataplane.GetListeners() {
		if _, ok := listeners[ln]; ok {
			// Listener exists in the resources fetched
			continue
		}
		f.logger.Debugf("Remove Listener: %s.", ln)
		f.dataplane.RemoveListener(ln)
	}
	return nil
}

func (f *fetcher) handleSecrets(resources []*anypb.Any) error {
	for _, res := range resources {
		secret := &tlsv3.Secret{}
		err := anypb.UnmarshalTo(res, secret, proto.UnmarshalOptions{})
		if err != nil {
			return err
		}
		f.logger.Debugf("Secret: %s.", secret.Name)
		if err := f.dataplane.AddSecret(secret); err != nil {
			return fmt.Errorf("error adding secret %s: %w", secret.Name, err)
		}
	}

	return nil
}

func (f *fetcher) Run() error {
	for {
		resp, err := f.client.Fetch()
		if err != nil {
			f.logger.Errorf("Failed to fetch %s: %v.", f.resourceType, err)
			return err
		}
		f.logger.Debugf("Fetched %s -> %+v", f.resourceType, resp.Resources)

		switch f.resourceType {
		case resource.ClusterType:
			err := f.handleClusters(resp.Resources)
			if err != nil {
				f.logger.Errorf("Failed to handle clusters: %v.", err)
			}
		case resource.ListenerType:
			err := f.handleListeners(resp.Resources)
			if err != nil {
				f.logger.Errorf("Failed to handle listeners: %v.", err)
			}
		case resource.SecretType:
			err := f.handleSecrets(resp.Resources)
			if err != nil {
				f.logger.Errorf("Failed to handle secrets: %v.", err)
			}
		default:
			return fmt.Errorf("unknown resource type")
		}

		err = f.client.Ack()
		if err != nil {
			f.logger.Errorf("Failed to ack: %v.", err)
		}
	}
}

func newFetcher(
	ctx context.Context,
	controlplaneClient grpc.ClientConnInterface,
	resourceType string,
	dp *server.Dataplane,
) (*fetcher, error) {
	cl := client.NewADSClient(ctx, &core.Node{Id: dp.ID}, resourceType)
	err := cl.InitConnect(controlplaneClient)
	if err != nil {
		return nil, err
	}
	return &fetcher{
		client:       cl,
		resourceType: resourceType,
		dataplane:    dp,
		logger:       logrus.WithField("component", "fetcher.xds.client"),
	}, nil
}
