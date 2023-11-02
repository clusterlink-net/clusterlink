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

package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"syscall"
	"time"

	"github.com/clusterlink-net/clusterlink/pkg/api"
	"github.com/clusterlink-net/clusterlink/pkg/client"
	"github.com/clusterlink-net/clusterlink/pkg/policyengine/policytypes"
)

// ClusterLink represents a clusterlink instance.
type ClusterLink struct {
	cluster   *KindCluster
	namespace string
	client    *client.Client
	port      uint16
}

// Name returns the peer name.
func (c *ClusterLink) Name() string {
	return c.cluster.Name()
}

// Namespace returns the clusterlink kubernetes namespace.
func (c *ClusterLink) Namespace() string {
	return c.namespace
}

// IP returns the peer IP.
func (c *ClusterLink) IP() string {
	return c.cluster.IP()
}

// Port returns the peer port.
func (c *ClusterLink) Port() uint16 {
	return c.port
}

// Cluster returns the backing kind cluster.
func (c *ClusterLink) Cluster() *KindCluster {
	return c.cluster
}

// WaitForControlplaneAPI waits until the controlplane API server is up.
func (c *ClusterLink) WaitForControlplaneAPI() error {
	var err error
	for t := time.Now(); time.Since(t) < time.Second*60; time.Sleep(time.Millisecond * 100) {
		_, err = c.client.Peers.List()
		switch {
		case err == nil:
			return nil
		case errors.Is(err, syscall.ECONNREFUSED):
			continue
		case errors.Is(err, syscall.ECONNRESET):
			continue
		case errors.Is(err, io.EOF):
			continue
		}

		return err
	}

	return err
}

// Access a cluster service.
func (c *ClusterLink) AccessService(service *Service, allowRetry bool) (string, error) {
	if service.Namespace == "" {
		service.Namespace = c.namespace
	}

	var data string
	var err error
	for t := time.Now(); time.Since(t) < time.Second*60; time.Sleep(time.Millisecond * 500) {
		data, err = c.cluster.AccessService(service)
		if !allowRetry {
			break
		}

		switch {
		case errors.Is(err, &ServiceNotFoundError{}):
			continue
		case errors.Is(err, &ConnectionRefusedError{}):
			continue
		case errors.Is(err, &ConnectionResetError{}):
			continue
		}

		break
	}

	return strings.TrimSpace(data), err
}

func (c *ClusterLink) CreatePeer(peer *ClusterLink) error {
	return c.client.Peers.Create(&api.Peer{
		Name: peer.Name(),
		Spec: api.PeerSpec{
			Gateways: []api.Endpoint{{
				Host: peer.IP(),
				Port: peer.Port(),
			}},
		},
	})
}

func (c *ClusterLink) UpdatePeer(peer *ClusterLink) error {
	return c.client.Peers.Update(&api.Peer{
		Name: peer.Name(),
		Spec: api.PeerSpec{
			Gateways: []api.Endpoint{{
				Host: peer.IP(),
				Port: peer.Port(),
			}},
		},
	})
}

func (c *ClusterLink) GetPeer(peer *ClusterLink) (*api.Peer, error) {
	res, err := c.client.Peers.Get(peer.Name())
	if err != nil {
		return nil, err
	}

	return res.(*api.Peer), nil
}

func (c *ClusterLink) GetAllPeers() (*[]api.Peer, error) {
	res, err := c.client.Peers.List()
	if err != nil {
		return nil, err
	}

	return res.(*[]api.Peer), nil
}

func (c *ClusterLink) DeletePeer(peer *ClusterLink) error {
	return c.client.Peers.Delete(peer.Name())
}

func (c *ClusterLink) CreateExport(name string, service *Service) error {
	return c.client.Exports.Create(&api.Export{
		Name: name,
		Spec: api.ExportSpec{
			Service: api.Endpoint{
				Host: fmt.Sprintf("%s.%s.svc.cluster.local", service.Name, service.Namespace),
				Port: service.Port,
			},
		},
	})
}

func (c *ClusterLink) UpdateExport(name string, service *Service) error {
	return c.client.Exports.Update(&api.Export{
		Name: name,
		Spec: api.ExportSpec{
			Service: api.Endpoint{
				Host: fmt.Sprintf("%s.%s.svc.cluster.local", service.Name, service.Namespace),
				Port: service.Port,
			},
		},
	})
}

func (c *ClusterLink) GetExport(name string) (*api.Export, error) {
	res, err := c.client.Exports.Get(name)
	if err != nil {
		return nil, err
	}

	return res.(*api.Export), nil
}

func (c *ClusterLink) GetAllExports() (*[]api.Export, error) {
	res, err := c.client.Exports.List()
	if err != nil {
		return nil, err
	}

	return res.(*[]api.Export), nil
}

func (c *ClusterLink) DeleteExport(name string) error {
	return c.client.Exports.Delete(name)
}

func (c *ClusterLink) CreateImport(name string, service *Service) error {
	return c.client.Imports.Create(&api.Import{
		Name: name,
		Spec: api.ImportSpec{
			Service: api.Endpoint{
				Host: service.Name,
				Port: service.Port,
			},
		},
	})
}

func (c *ClusterLink) UpdateImport(name string, service *Service) error {
	return c.client.Imports.Update(&api.Import{
		Name: name,
		Spec: api.ImportSpec{
			Service: api.Endpoint{
				Host: service.Name,
				Port: service.Port,
			},
		},
	})
}

func (c *ClusterLink) GetImport(name string) (*api.Import, error) {
	res, err := c.client.Imports.Get(name)
	if err != nil {
		return nil, err
	}

	return res.(*api.Import), nil
}

func (c *ClusterLink) GetAllImports() (*[]api.Import, error) {
	res, err := c.client.Imports.List()
	if err != nil {
		return nil, err
	}

	return res.(*[]api.Import), nil
}

func (c *ClusterLink) DeleteImport(name string) error {
	return c.client.Imports.Delete(name)
}

func (c *ClusterLink) CreateBinding(imp string, peer *ClusterLink) error {
	return c.client.Bindings.Create(&api.Binding{
		Spec: api.BindingSpec{
			Import: imp,
			Peer:   peer.Name(),
		},
	})
}

func (c *ClusterLink) UpdateBinding(imp string, peer *ClusterLink) error {
	return c.client.Bindings.Update(&api.Binding{
		Spec: api.BindingSpec{
			Import: imp,
			Peer:   peer.Name(),
		},
	})
}

func (c *ClusterLink) GetBindings(name string) (*[]api.Binding, error) {
	res, err := c.client.Bindings.Get(name)
	if err != nil {
		return nil, err
	}

	return res.(*[]api.Binding), nil
}

func (c *ClusterLink) GetAllBindings() (*[]api.Binding, error) {
	res, err := c.client.Bindings.List()
	if err != nil {
		return nil, err
	}

	return res.(*[]api.Binding), nil
}

func (c *ClusterLink) DeleteBinding(imp string, peer *ClusterLink) error {
	return c.client.Bindings.Delete(&api.Binding{
		Spec: api.BindingSpec{
			Import: imp,
			Peer:   peer.Name(),
		},
	})
}

func (c *ClusterLink) CreatePolicy(policy *policytypes.ConnectivityPolicy) error {
	data, err := json.Marshal(policy)
	if err != nil {
		return err
	}

	return c.client.AccessPolicies.Create(&api.Policy{
		Name: policy.Name,
		Spec: api.PolicySpec{Blob: data},
	})
}

func (c *ClusterLink) UpdatePolicy(policy *policytypes.ConnectivityPolicy) error {
	data, err := json.Marshal(policy)
	if err != nil {
		return err
	}

	return c.client.AccessPolicies.Update(&api.Policy{
		Name: policy.Name,
		Spec: api.PolicySpec{Blob: data},
	})
}

func (c *ClusterLink) GetPolicy(name string) (*policytypes.ConnectivityPolicy, error) {
	res, err := c.client.AccessPolicies.Get(name)
	if err != nil {
		return nil, err
	}

	var policy policytypes.ConnectivityPolicy
	if err := json.Unmarshal(res.(*api.Policy).Spec.Blob, &policy); err != nil {
		return nil, err
	}

	return &policy, nil
}

func (c *ClusterLink) GetAllPolicies() (*[]policytypes.ConnectivityPolicy, error) {
	res, err := c.client.AccessPolicies.List()
	if err != nil {
		return nil, err
	}

	policies := make([]policytypes.ConnectivityPolicy, len(*res.(*[]api.Policy)))
	for i, p := range *res.(*[]api.Policy) {
		if err := json.Unmarshal(p.Spec.Blob, &policies[i]); err != nil {
			return nil, err
		}
	}

	return &policies, nil
}

func (c *ClusterLink) DeletePolicy(name string) error {
	return c.client.AccessPolicies.Delete(name)
}
