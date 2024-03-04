// Copyright 2023 The ClusterLink Authors.
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

package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"

	"github.com/clusterlink-net/clusterlink/pkg/controlplane/api"
	utiltls "github.com/clusterlink-net/clusterlink/pkg/util/tls"
)

// Dataplane implements the server and api client which sends authorization to the control plane.
type Dataplane struct {
	ID                 string
	peerName           string
	router             *chi.Mux
	apiClient          *http.Client
	parsedCertData     *utiltls.ParsedCertData
	controlplaneTarget string
	clusters           map[string]*cluster.Cluster
	listeners          map[string]*listener.Listener
	listenerEnd        map[string]chan bool
	clustersLock       sync.Mutex
	listenerLock       sync.Mutex
	logger             *logrus.Entry
}

// GetClusterTarget returns the cluster address:port from the cluster map.
func (d *Dataplane) GetClusterTarget(name string) (string, error) {
	if _, ok := d.clusters[name]; !ok {
		return "", fmt.Errorf("unable to find %s in cluster map", name)
	}
	ep := d.clusters[name].LoadAssignment.GetEndpoints()[0].LbEndpoints[0].GetEndpoint().Address.GetSocketAddress()
	return ep.GetAddress() + ":" + strconv.Itoa(int(ep.GetPortValue())), nil
}

// GetClusterHost returns the cluster hostname after trimming ":".
func (d *Dataplane) GetClusterHost(name string) (string, error) {
	if _, ok := d.clusters[name]; !ok {
		return "", fmt.Errorf("unable to find %s in cluster map", name)
	}
	return strings.Split(
		d.clusters[name].LoadAssignment.GetEndpoints()[0].LbEndpoints[0].GetEndpoint().Hostname, ":")[0], nil
}

// AddCluster adds/updates a cluster to the map.
func (d *Dataplane) AddCluster(c *cluster.Cluster) {
	d.clustersLock.Lock()
	d.clusters[c.Name] = c
	d.clustersLock.Unlock()
}

// RemoveCluster adds a cluster to the map.
func (d *Dataplane) RemoveCluster(name string) {
	d.clustersLock.Lock()
	delete(d.clusters, name)
	d.clustersLock.Unlock()
}

// GetClusters returns the listeners map.
func (d *Dataplane) GetClusters() map[string]*cluster.Cluster {
	return d.clusters
}

// AddListener adds a listener to the map.
func (d *Dataplane) AddListener(ln *listener.Listener) {
	listenerName := strings.TrimPrefix(ln.Name, api.ImportListenerPrefix)
	if le, ok := d.listeners[listenerName]; ok {
		// Check if there is an update to the listener address/port
		if ln.Address.GetSocketAddress().GetAddress() == le.Address.GetSocketAddress().GetAddress() &&
			ln.Address.GetSocketAddress().GetPortValue() == le.Address.GetSocketAddress().GetPortValue() {
			return
		}
		d.listenerEnd[listenerName] <- true
	}
	d.listenerLock.Lock()
	d.listeners[listenerName] = ln
	d.listenerLock.Unlock()
	go func() {
		d.CreateListener(listenerName,
			ln.Address.GetSocketAddress().GetAddress(),
			ln.Address.GetSocketAddress().GetPortValue())
	}()
}

// RemoveListener removes a listener.
func (d *Dataplane) RemoveListener(name string) {
	d.listenerLock.Lock()
	delete(d.listeners, name)
	d.listenerLock.Unlock()
	d.listenerEnd[name] <- true
}

// GetListeners returns the listeners map.
func (d *Dataplane) GetListeners() map[string]*listener.Listener {
	return d.listeners
}

// NewDataplane returns a new dataplane HTTP server.
func NewDataplane(dataplaneID, controlplaneTarget, peerName string, parsedCertData *utiltls.ParsedCertData) *Dataplane {
	dp := &Dataplane{
		ID:       dataplaneID,
		peerName: peerName,
		router:   chi.NewRouter(),
		apiClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: parsedCertData.ClientConfig(peerName),
			},
		},
		parsedCertData:     parsedCertData,
		controlplaneTarget: controlplaneTarget,
		clusters:           make(map[string]*cluster.Cluster),
		listeners:          make(map[string]*listener.Listener),
		listenerEnd:        make(map[string]chan bool),
		logger:             logrus.WithField("component", "dataplane.server.http"),
	}

	dp.addAuthzHandlers()
	return dp
}
