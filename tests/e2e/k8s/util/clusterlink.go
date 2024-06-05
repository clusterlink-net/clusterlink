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

package util

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"

	"github.com/clusterlink-net/clusterlink/cmd/cl-controlplane/app"
	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	"github.com/clusterlink-net/clusterlink/pkg/bootstrap"
	"github.com/clusterlink-net/clusterlink/tests/e2e/k8s/services"
)

// ClusterLink represents a clusterlink instance.
type ClusterLink struct {
	cluster   *KindCluster
	namespace string
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

// ScaleControlplane scales the controlplane deployment.
func (c *ClusterLink) ScaleControlplane(replicas int32) error {
	return c.cluster.ScaleDeployment("cl-controlplane", c.namespace, replicas)
}

// RestartControlplane restarts the controlplane.
func (c *ClusterLink) RestartControlplane() error {
	if err := c.ScaleControlplane(0); err != nil {
		return err
	}
	return c.ScaleControlplane(1)
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

// RestartDataplane restarts the dataplane.
func (c *ClusterLink) UpdatePeerCertificates(
	fabricCert *bootstrap.Certificate, peerCert *bootstrap.Certificate,
) error {
	err := c.cluster.Resources().Update(
		context.Background(),
		&v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cl-peer",
				Namespace: c.namespace,
			},
			Data: map[string][]byte{
				app.PeerCertificateFile:   peerCert.RawCert(),
				app.PeerKeyFile:           peerCert.RawKey(),
				app.FabricCertificateFile: fabricCert.RawCert(),
			},
		})
	if err != nil {
		return fmt.Errorf("cannot update peer secret: %w", err)
	}

	// update controlplane pods annotation to speed-up re-loading of secret
	var pods v1.PodList
	err = c.cluster.Resources().List(
		context.Background(),
		&pods,
		resources.WithLabelSelector("app=cl-controlplane"))
	if err != nil {
		return fmt.Errorf("unable to list controlplane pods: %w", err)
	}

	mergePatch, err := json.Marshal(map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": map[string]interface{}{
				"peer-tls-last-updated": time.Now().String(),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("cannot encode pod annotation patch: %w", err)
	}

	for i := range pods.Items {
		err := c.cluster.Resources().Patch(
			context.Background(),
			&pods.Items[i],
			k8s.Patch{
				PatchType: types.StrategicMergePatchType,
				Data:      mergePatch,
			},
		)
		if err != nil {
			return fmt.Errorf("unable to annotate controlplane pod: %w", err)
		}
	}

	return nil
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
		case errors.Is(err, &PodFailedError{}):
			continue
		case err == nil && expectedError != nil:
			continue
		}

		break
	}

	return data, err
}

func (c *ClusterLink) CreatePeer(peer *ClusterLink) error {
	pr := &v1alpha1.Peer{
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
	}

	return c.cluster.Resources().Create(context.Background(), pr)
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

func (c *ClusterLink) DeleteService(name string) error {
	return c.cluster.Resources().Delete(
		context.Background(),
		&v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: c.namespace,
			},
		})
}

func (c *ClusterLink) CreateExport(service *Service) error {
	export := &v1alpha1.Export{
		ObjectMeta: metav1.ObjectMeta{
			Name:      service.Name,
			Namespace: c.namespace,
			Labels:    service.Labels,
		},
		Spec: v1alpha1.ExportSpec{
			Port: service.Port,
		},
	}

	return c.cluster.Resources().Create(context.Background(), export)
}

func (c *ClusterLink) DeleteExport(name string) error {
	return c.cluster.Resources().Delete(
		context.Background(),
		&v1alpha1.Export{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: c.namespace,
			},
		})
}

func (c *ClusterLink) CreateImport(service *Service, peer *ClusterLink, exportName string) error {
	imp := &v1alpha1.Import{
		ObjectMeta: metav1.ObjectMeta{
			Name:      service.Name,
			Namespace: c.namespace,
			Labels:    service.Labels,
		},
		Spec: v1alpha1.ImportSpec{
			Port: service.Port,
			Sources: []v1alpha1.ImportSource{{
				Peer:            peer.Name(),
				ExportName:      exportName,
				ExportNamespace: peer.Namespace(),
			}},
		},
	}

	return c.cluster.Resources().Create(context.Background(), imp)
}

func (c *ClusterLink) UpdateImport(service *Service, peer *ClusterLink, exportName string) error {
	imp := &v1alpha1.Import{
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
	}

	return c.cluster.Resources().Update(context.Background(), imp)
}

func (c *ClusterLink) CreatePolicy(policy *v1alpha1.AccessPolicy) error {
	if policy.Namespace == "" {
		accessPolicyCopy := *policy
		accessPolicyCopy.Namespace = c.namespace
		policy = &accessPolicyCopy
	}

	return c.cluster.Resources().Create(context.Background(), policy)
}

func (c *ClusterLink) DeletePolicy(name string) error {
	return c.cluster.Resources().Delete(
		context.Background(),
		&v1alpha1.AccessPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: c.namespace,
			},
		})
}

func (c *ClusterLink) CreatePrivilegedPolicy(policy *v1alpha1.PrivilegedAccessPolicy) error {
	return c.cluster.Resources().Create(context.Background(), policy)
}

func (c *ClusterLink) DeletePrivilegedPolicy(name string) error {
	return c.cluster.Resources().Delete(
		context.Background(),
		&v1alpha1.PrivilegedAccessPolicy{
			ObjectMeta: metav1.ObjectMeta{Name: name},
		})
}

func (c *ClusterLink) WaitForImportCondition(
	imp *v1alpha1.Import,
	conditionType string,
	expectedConditionStatus bool,
) error {
	return c.cluster.WaitFor(imp, &imp.Status.Conditions, conditionType, expectedConditionStatus)
}

func (c *ClusterLink) WaitForExportCondition(
	export *v1alpha1.Export,
	conditionType string,
	expectedConditionStatus bool,
) error {
	return c.cluster.WaitFor(export, &export.Status.Conditions, conditionType, expectedConditionStatus)
}

func (c *ClusterLink) WaitForPeerCondition(
	peer *v1alpha1.Peer,
	conditionType string,
	expectedConditionStatus bool,
) error {
	return c.cluster.WaitFor(peer, &peer.Status.Conditions, conditionType, expectedConditionStatus)
}
