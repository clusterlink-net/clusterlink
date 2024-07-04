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
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/clusterlink-net/clusterlink/pkg/dataplane/server"
)

// resources indicate the xDS resources that would be fetched.
var resources = [...]string{resource.ClusterType, resource.ListenerType, resource.SecretType}

// XDSClient implements the client which fetches clusters and listeners.
type XDSClient struct {
	dataplane          *server.Dataplane
	controlplaneClient grpc.ClientConnInterface
	lock               sync.Mutex
	errors             map[string]error
	logger             *logrus.Entry
	clustersReady      chan bool
	fetchers           map[string]*fetcher
}

func (x *XDSClient) runFetcher(fetcher *fetcher) error {
	for {
		for {
			err := fetcher.client.InitConnect(x.controlplaneClient)
			if err == nil {
				break
			}

			x.logger.Errorf("Failed to initialize %s fetcher: %v.", fetcher.resourceType, err)
			time.Sleep(time.Second)
		}

		// If the resource type is listener, it shouldn't run until the cluster fetcher is running
		switch fetcher.resourceType {
		case resource.ClusterType:
			x.clustersReady <- true
		case resource.ListenerType:
			<-x.clustersReady
			x.logger.Infof("Done waiting for cluster fetcher")
		}
		x.logger.Infof("Starting to run %s fetcher.", fetcher.resourceType)
		err := fetcher.Run()
		x.logger.Infof("Fetcher '%s' stopped: %v.", fetcher.resourceType, err)
	}
}

func (x *XDSClient) IsReady() bool {
	for _, fetcher := range x.fetchers {
		if !fetcher.IsReady() {
			return false
		}
	}
	return true
}

// Run starts the running xDS client which fetches clusters and listeners from the controlplane.
func (x *XDSClient) Run() error {
	var wg sync.WaitGroup

	wg.Add(len(resources))
	for _, res := range resources {
		go func(res string) {
			defer wg.Done()
			err := x.runFetcher(x.fetchers[res])
			x.logger.Errorf("Fetcher (%s) stopped: %v", res, err)

			x.lock.Lock()
			x.errors[res] = err
			x.lock.Unlock()
		}(res)
	}
	wg.Wait()

	var errs []error
	for resource, err := range x.errors {
		if err != nil {
			errs = append(errs, fmt.Errorf(
				"error running fetcher '%s': %w", resource, err))
		}
	}

	return errors.Join(errs...)
}

// NewXDSClient returns am xDS client which can fetch clusters and listeners from the controlplane.
func NewXDSClient(dataplane *server.Dataplane, controlplaneClient grpc.ClientConnInterface) *XDSClient {
	fetchers := make(map[string]*fetcher, len(resources))
	for _, res := range resources {
		fetchers[res] = newFetcher(context.Background(), res, dataplane)
	}

	return &XDSClient{
		dataplane:          dataplane,
		controlplaneClient: controlplaneClient,
		errors:             make(map[string]error),
		logger:             logrus.WithField("component", "xds.client"),
		clustersReady:      make(chan bool, 1),
		fetchers:           fetchers,
	}
}
