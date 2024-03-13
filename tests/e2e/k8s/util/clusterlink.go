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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"syscall"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/clusterlink-net/clusterlink/pkg/api"
	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	"github.com/clusterlink-net/clusterlink/pkg/client"
	"github.com/clusterlink-net/clusterlink/pkg/policyengine/policytypes"
	"github.com/clusterlink-net/clusterlink/tests/e2e/k8s/services"
)

// ClusterLink represents a clusterlink instance.
type ClusterLink struct {
	cluster   *KindCluster
	namespace string
	client    *client.Client
	port      uint16
	crdMode   bool
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

// Client returns a controlplane API client for this cluster.
func (c *ClusterLink) Client() *client.Client {
	return c.client
}

// WaitForControlplaneAPI waits until the controlplane API server is up.
func (c *ClusterLink) WaitForControlplaneAPI() error {
	var err error
	for t := time.Now(); time.Since(t) < time.Second*60; time.Sleep(time.Millisecond * 100) {
		var uerr *url.Error
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
		case errors.As(err, &uerr) && uerr.Timeout():
			continue
		}

		return err
	}

	return err
}

// ScaleControlplane scales the controlplane deployment.
func (c *ClusterLink) ScaleControlplane(replicas int32) error {
	return c.cluster.ScaleDeployment("cl-controlplane", c.namespace, replicas)
}

// RestartControlplane restarts the controlplane.
func (c *ClusterLink) RestartControlplane() error {
	if err := c.ScaleControlplane(0); err != nil {
		return err
	}
	if err := c.ScaleControlplane(1); err != nil {
		return err
	}
	return c.WaitForControlplaneAPI()
}

// ScaleDataplane scales the dataplane deployment.
func (c *ClusterLink) ScaleDataplane(replicas int32) error {
	return c.cluster.ScaleDeployment("cl-dataplane", c.namespace, replicas)
}

// RestartDataplane restarts the dataplane.
func (c *ClusterLink) RestartDataplane() error {
	if err := c.ScaleDataplane(0); err != nil {
		return err
	}
	return c.ScaleDataplane(1)
}

// Access a cluster service.
func (c *ClusterLink) AccessService(
	clientFn func(*KindCluster, *Service) (string, error),
	service *Service, allowRetry bool, expectedError error,
) (string, error) {
	if service.Namespace == "" {
		service.Namespace = c.namespace
	}

	var data string
	var err error
	for t := time.Now(); time.Since(t) < time.Second*60; time.Sleep(time.Millisecond * 500) {
		data, err = clientFn(c.cluster, service)
		if errors.Is(err, expectedError) || !allowRetry {
			break
		}

		switch {
		case errors.Is(err, &services.ServiceNotFoundError{}):
			continue
		case errors.Is(err, &services.ConnectionRefusedError{}):
			continue
		case errors.Is(err, &services.ConnectionResetError{}):
			continue
		case err == nil && expectedError != nil:
			continue
		}

		break
	}

	return data, err
}

func (c *ClusterLink) CreatePeer(peer *ClusterLink) error {
	if c.crdMode {
		return c.cluster.Resources().Create(
			context.Background(),
			&v1alpha1.Peer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      peer.Name(),
					Namespace: c.namespace,
				},
				Spec: v1alpha1.PeerSpec{
					Gateways: []v1alpha1.Endpoint{{
						Host: peer.IP(),
						Port: peer.Port(),
					}},
				},
			})
	}

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

func (c *ClusterLink) CreateService(service *Service) error {
	return c.cluster.Resources().Create(
		context.Background(),
		&v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      service.Name,
				Namespace: c.namespace,
			},
			Spec: v1.ServiceSpec{
				Type:         v1.ServiceTypeExternalName,
				ExternalName: fmt.Sprintf("%s.%s.svc.cluster.local", service.Name, service.Namespace),
			},
		})
}

func (c *ClusterLink) CreateExport(name string, service *Service) error {
	if c.crdMode {
		return c.cluster.Resources().Create(
			context.Background(),
			&v1alpha1.Export{
				ObjectMeta: metav1.ObjectMeta{
					Name:      service.Name,
					Namespace: c.namespace,
				},
				Spec: v1alpha1.ExportSpec{
					Port: service.Port,
				},
			})
	}

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

func (c *ClusterLink) CreateImport(service *Service, peer *ClusterLink, exportName string) error {
	if c.crdMode {
		return c.cluster.Resources().Create(
			context.Background(),
			&v1alpha1.Import{
				ObjectMeta: metav1.ObjectMeta{
					Name:      service.Name,
					Namespace: c.namespace,
				},
				Spec: v1alpha1.ImportSpec{
					Port: service.Port,
					Sources: []v1alpha1.ImportSource{{
						Peer:            peer.Name(),
						ExportName:      exportName,
						ExportNamespace: peer.Namespace(),
					}},
				},
			})
	}

	return c.client.Imports.Create(&api.Import{
		Name: service.Name,
		Spec: api.ImportSpec{
			Port:  service.Port,
			Peers: []string{peer.Name()},
		},
	})
}

func (c *ClusterLink) UpdateImport(service *Service, peer *ClusterLink, exportName string) error {
	if c.crdMode {
		return c.cluster.Resources().Update(
			context.Background(),
			&v1alpha1.Import{
				ObjectMeta: metav1.ObjectMeta{
					Name:      service.Name,
					Namespace: c.namespace,
				},
				Spec: v1alpha1.ImportSpec{
					Port: service.Port,
					Sources: []v1alpha1.ImportSource{{
						Peer:            peer.Name(),
						ExportName:      exportName,
						ExportNamespace: peer.Namespace(),
					}},
				},
			})
	}

	return c.client.Imports.Update(&api.Import{
		Name: service.Name,
		Spec: api.ImportSpec{
			Port:  service.Port,
			Peers: []string{peer.Name()},
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

func (c *ClusterLink) CreateAccessPolicy(accessPolicy *v1alpha1.AccessPolicy) error {
	if accessPolicy.Namespace == "" {
		accessPolicyCopy := *accessPolicy
		accessPolicyCopy.Namespace = c.namespace
		accessPolicy = &accessPolicyCopy
	}
	return c.cluster.Resources().Create(context.Background(), accessPolicy)
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
