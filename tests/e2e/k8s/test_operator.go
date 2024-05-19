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

// Copyright (c) 2022 The ClusterLink Authors.
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

// Copyright (C) The ClusterLink Authors.
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

package k8s

import (
	"context"
	"time"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiwait "k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/e2e-framework/klient/wait"

	clusterlink "github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	"github.com/clusterlink-net/clusterlink/pkg/bootstrap/platform"
	"github.com/clusterlink-net/clusterlink/pkg/operator/controller"
	"github.com/clusterlink-net/clusterlink/tests/e2e/k8s/services"
	"github.com/clusterlink-net/clusterlink/tests/e2e/k8s/services/httpecho"
	"github.com/clusterlink-net/clusterlink/tests/e2e/k8s/util"
)

// TestOperator test the operator functionality.
func (s *TestSuite) TestOperator() {
	// Deploy ClusterLink with operator
	cfg := &util.PeerConfig{
		CRUDMode:           false,
		DataplaneType:      platform.DataplaneTypeEnvoy,
		Dataplanes:         1,
		DeployWithOperator: true,
	}
	cl, err := s.fabric.DeployClusterlinks(2, cfg)
	require.Nil(s.T(), err)

	// Deploy instance2
	instance2 := &clusterlink.Instance{
		ObjectMeta: v1.ObjectMeta{
			Name:      "instance2",
			Namespace: controller.OperatorNamespace,
		},
		Spec: clusterlink.InstanceSpec{
			Namespace:         s.fabric.Namespace(),
			ContainerRegistry: "docker.io/library", // Tell kind to use local image.
			Ingress:           clusterlink.IngressSpec{Type: "NodePort", Port: int32(cl[0].Port())},
		},
	}

	peerResource := s.fabric.PeerKindCluster(0).Resources()
	err = peerResource.Create(context.Background(), instance2)

	// Check basic connectivity with instances deployed by operator
	require.Nil(s.T(), err)
	require.Nil(s.T(), cl[0].CreateService(&httpEchoService))
	require.Nil(s.T(), cl[0].CreateExport(&httpEchoService))
	require.Nil(s.T(), cl[0].CreatePolicy(util.PolicyAllowAll))
	require.Nil(s.T(), cl[1].CreatePeer(cl[0]))

	importedService := &util.Service{
		Name: httpEchoService.Name,
		Port: 80,
	}

	require.Nil(s.T(), cl[1].CreateImport(importedService, cl[0], httpEchoService.Name))
	require.Nil(s.T(), cl[1].CreatePolicy(util.PolicyAllowAll))

	data, err := cl[1].AccessService(httpecho.GetEchoValue, importedService, true, nil)
	require.Nil(s.T(), err)
	require.Equal(s.T(), cl[0].Name(), data)

	// Verify that instance2 failed.
	instanceReadyCondition := func(instance *clusterlink.Instance, condStatus v1.ConditionStatus) apiwait.ConditionWithContextFunc {
		return func(ctx context.Context) (bool, error) {
			done := false
			if err := peerResource.Get(ctx, instance.GetName(), instance.GetNamespace(), instance); err != nil {
				return false, err
			}
			if c, ok := instance.Status.Controlplane.Conditions[string(clusterlink.DeploymentReady)]; ok {
				if c.Status == condStatus {
					done = true
				}
			}
			return done, nil
		}
	}

	// Check that instance1 deployment succeeded.
	instance1 := &clusterlink.Instance{
		ObjectMeta: v1.ObjectMeta{
			Name:      "cl-instance" + s.fabric.Namespace(),
			Namespace: controller.OperatorNamespace,
		},
	}
	err = wait.For(instanceReadyCondition(instance1, v1.ConditionTrue), wait.WithTimeout(time.Second*60))
	require.Nil(s.T(), err)
	err = peerResource.Get(context.Background(), "cl-instance"+s.fabric.Namespace(), controller.OperatorNamespace, instance1)
	require.Nil(s.T(), err)
	require.Equal(s.T(), v1.ConditionTrue, instance1.Status.Dataplane.Conditions[string(clusterlink.DeploymentReady)].Status)
	require.Equal(s.T(), v1.ConditionTrue, instance1.Status.Ingress.Conditions[string(clusterlink.ServiceReady)].Status)

	err = wait.For(instanceReadyCondition(instance2, v1.ConditionFalse), wait.WithTimeout(time.Second*60))
	require.Nil(s.T(), err)
	err = peerResource.Get(context.Background(), "instance2", controller.OperatorNamespace, instance2)
	require.Nil(s.T(), err)
	require.Equal(s.T(), v1.ConditionFalse, instance2.Status.Controlplane.Conditions[string(clusterlink.DeploymentReady)].Status)
	require.Equal(s.T(), v1.ConditionFalse, instance2.Status.Dataplane.Conditions[string(clusterlink.DeploymentReady)].Status)

	// Delete first instance.
	err = peerResource.Delete(context.Background(), instance1)
	require.Nil(s.T(), err)
	// Check failure to access service after deletion
	_, err = cl[1].AccessService(httpecho.GetEchoValue, importedService, true, &services.ConnectionResetError{})
	require.Equal(s.T(), &services.ConnectionResetError{}, err)
	// Check that instance2 succeeded.
	err = wait.For(instanceReadyCondition(instance2, v1.ConditionTrue), wait.WithTimeout(time.Second*60))
	require.Nil(s.T(), err)
	err = peerResource.Get(context.Background(), "instance2", controller.OperatorNamespace, instance2)
	require.Nil(s.T(), err)
	require.Equal(s.T(), v1.ConditionTrue, instance2.Status.Controlplane.Conditions[string(clusterlink.DeploymentReady)].Status)
	require.Equal(s.T(), v1.ConditionTrue, instance2.Status.Dataplane.Conditions[string(clusterlink.DeploymentReady)].Status)
	require.Equal(s.T(), v1.ConditionTrue, instance2.Status.Ingress.Conditions[string(clusterlink.ServiceReady)].Status)

	// Check access service in the new instance
	data, err = cl[1].AccessService(httpecho.GetEchoValue, importedService, true, nil)
	require.Nil(s.T(), err)
	require.Equal(s.T(), cl[0].Name(), data)
}
