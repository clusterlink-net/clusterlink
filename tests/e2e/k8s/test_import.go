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

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/control"
	"github.com/clusterlink-net/clusterlink/tests/e2e/k8s/services"
	"github.com/clusterlink-net/clusterlink/tests/e2e/k8s/services/httpecho"
	"github.com/clusterlink-net/clusterlink/tests/e2e/k8s/util"
)

func (s *TestSuite) TestImportConflictingTargetPort() {
	cl, err := s.fabric.DeployClusterlinks(1, nil)
	require.Nil(s.T(), err)

	imp1 := &v1alpha1.Import{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "imp1",
			Namespace: cl[0].Namespace(),
		},
		Spec: v1alpha1.ImportSpec{
			Port:       80,
			TargetPort: 1234,
			Sources:    []v1alpha1.ImportSource{{}},
		},
	}

	require.Nil(s.T(), cl[0].Cluster().Resources().Create(context.Background(), imp1))
	require.Nil(s.T(), cl[0].WaitForImportCondition(imp1, v1alpha1.ImportServiceCreated, true))

	imp2 := &v1alpha1.Import{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "imp2",
			Namespace: cl[0].Namespace(),
		},
		Spec: v1alpha1.ImportSpec{
			Port:       80,
			TargetPort: 1234,
			Sources:    []v1alpha1.ImportSource{},
		},
	}

	require.Nil(s.T(), cl[0].Cluster().Resources().Create(context.Background(), imp2))
	require.Nil(s.T(), cl[0].WaitForImportCondition(imp2, v1alpha1.ImportServiceCreated, false))
	require.True(s.T(), meta.IsStatusConditionFalse(imp2.Status.Conditions, v1alpha1.ImportTargetPortValid))

	imp2Service := &util.Service{
		Name: imp2.Name,
		Port: imp2.Spec.Port,
	}

	_, err = cl[0].AccessService(httpecho.GetEchoValue, imp2Service, true, &services.ServiceNotFoundError{})
	require.ErrorIs(s.T(), err, &services.ServiceNotFoundError{})

	imp2.Spec.TargetPort = 1235
	require.Nil(s.T(), cl[0].Cluster().Resources().Update(context.Background(), imp2))
	require.Nil(s.T(), cl[0].WaitForImportCondition(imp2, v1alpha1.ImportServiceCreated, true))
	require.True(s.T(), meta.IsStatusConditionTrue(imp2.Status.Conditions, v1alpha1.ImportTargetPortValid))

	_, err = cl[0].AccessService(httpecho.GetEchoValue, imp2Service, true, &services.ConnectionResetError{})
	require.ErrorIs(s.T(), err, &services.ConnectionResetError{})
}

func (s *TestSuite) TestImportConflictingService() {
	cl, err := s.fabric.DeployClusterlinks(1, nil)
	require.Nil(s.T(), err)

	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "service",
			Namespace: cl[0].Namespace(),
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{{
				Port: 80,
			}},
		},
	}

	require.Nil(s.T(), cl[0].Cluster().Resources().Create(context.Background(), service))

	imp := &v1alpha1.Import{
		ObjectMeta: metav1.ObjectMeta{
			Name:      service.Name,
			Namespace: cl[0].Namespace(),
		},
		Spec: v1alpha1.ImportSpec{
			Port:    80,
			Sources: []v1alpha1.ImportSource{},
		},
	}

	require.Nil(s.T(), cl[0].Cluster().Resources().Create(context.Background(), imp))
	require.Nil(s.T(), cl[0].WaitForImportCondition(imp, v1alpha1.ImportServiceCreated, false))

	service.Labels = make(map[string]string)
	service.Labels[control.LabelManagedBy] = control.AppName
	service.Labels[control.LabelImportName] = imp.Name
	service.Labels[control.LabelImportNamespace] = imp.Namespace
	require.Nil(s.T(), cl[0].Cluster().Resources().Update(context.Background(), service))
	require.Nil(s.T(), cl[0].WaitForImportCondition(imp, v1alpha1.ImportServiceCreated, true))

	impService := &util.Service{
		Name: imp.Name,
		Port: imp.Spec.Port,
	}

	_, err = cl[0].AccessService(httpecho.GetEchoValue, impService, true, &services.ConnectionResetError{})
	require.ErrorIs(s.T(), err, &services.ConnectionResetError{})

	require.Nil(s.T(), cl[0].DeleteService(service.Name))
	require.Nil(s.T(), cl[0].WaitForImportCondition(imp, v1alpha1.ImportServiceCreated, true))
	_, err = cl[0].AccessService(httpecho.GetEchoValue, impService, true, &services.ConnectionResetError{})
	require.ErrorIs(s.T(), err, &services.ConnectionResetError{})

	require.Nil(s.T(), cl[0].Cluster().Resources().Get(context.Background(), service.Name, service.Namespace, service))
	service.Labels[control.LabelManagedBy] = "other"
	require.Nil(s.T(), cl[0].Cluster().Resources().Update(context.Background(), service))
	require.Nil(s.T(), cl[0].WaitForImportCondition(imp, v1alpha1.ImportServiceCreated, false))

	service.Labels[control.LabelManagedBy] = control.AppName
	require.Nil(s.T(), cl[0].Cluster().Resources().Update(context.Background(), service))
	require.Nil(s.T(), cl[0].WaitForImportCondition(imp, v1alpha1.ImportServiceCreated, true))
	_, err = cl[0].AccessService(httpecho.GetEchoValue, impService, true, &services.ConnectionResetError{})
	require.ErrorIs(s.T(), err, &services.ConnectionResetError{})
}

func (s *TestSuite) TestImportUnprivilegedNamespace() {
	cl, err := s.fabric.DeployClusterlinks(1, nil)
	require.Nil(s.T(), err)

	namespace := cl[0].Namespace() + "-unprivileged"

	require.Nil(s.T(), cl[0].Cluster().CreateNamespace(namespace))
	defer func() {
		require.Nil(s.T(), cl[0].Cluster().DeleteNamespace(namespace))
	}()

	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "service",
			Namespace: namespace,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{{
				Port: 80,
			}},
		},
	}

	systemService := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: control.SystemServiceName(types.NamespacedName{
				Namespace: service.Namespace,
				Name:      service.Name,
			}),
			Namespace: cl[0].Namespace(),
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{{
				Port: 80,
			}},
		},
	}

	require.Nil(s.T(), cl[0].Cluster().Resources().Create(context.Background(), systemService))

	imp := &v1alpha1.Import{
		ObjectMeta: metav1.ObjectMeta{
			Name:      service.Name,
			Namespace: namespace,
		},
		Spec: v1alpha1.ImportSpec{
			Port:    80,
			Sources: []v1alpha1.ImportSource{},
		},
	}

	require.Nil(s.T(), cl[0].Cluster().Resources().Create(context.Background(), imp))
	require.Nil(s.T(), cl[0].WaitForImportCondition(imp, v1alpha1.ImportServiceCreated, false))

	require.Nil(s.T(), cl[0].Cluster().Resources().Delete(context.Background(), systemService))
	require.Nil(s.T(), cl[0].WaitForImportCondition(imp, v1alpha1.ImportServiceCreated, true))

	require.Nil(s.T(), cl[0].Cluster().Resources().Get(context.Background(), service.Name, service.Namespace, service))
	service.Labels[control.LabelManagedBy] = "other"
	service.Spec.ExternalName = "broken"
	require.Nil(s.T(), cl[0].Cluster().Resources().Update(context.Background(), service))
	require.Nil(s.T(), cl[0].WaitForImportCondition(imp, v1alpha1.ImportServiceCreated, false))

	service.Labels[control.LabelManagedBy] = control.AppName
	require.Nil(s.T(), cl[0].Cluster().Resources().Update(context.Background(), service))
	require.Nil(s.T(), cl[0].WaitForImportCondition(imp, v1alpha1.ImportServiceCreated, true))

	impService := &util.Service{
		Name:      imp.Name,
		Namespace: namespace,
		Port:      imp.Spec.Port,
	}

	_, err = cl[0].AccessService(httpecho.GetEchoValue, impService, true, &services.ConnectionResetError{})
	require.ErrorIs(s.T(), err, &services.ConnectionResetError{})

	require.Nil(s.T(), cl[0].Cluster().Resources().Delete(context.Background(), imp))
	require.Nil(s.T(), cl[0].Cluster().WaitForDeletion(service))
	require.Nil(s.T(), cl[0].Cluster().WaitForDeletion(systemService))
}

func (s *TestSuite) TestImportDelete() {
	cl, err := s.fabric.DeployClusterlinks(1, nil)
	require.Nil(s.T(), err)

	imp := &v1alpha1.Import{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "imp",
			Namespace: cl[0].Namespace(),
		},
		Spec: v1alpha1.ImportSpec{
			Port:       80,
			TargetPort: 1234,
			Sources:    []v1alpha1.ImportSource{{}},
		},
	}

	require.Nil(s.T(), cl[0].Cluster().Resources().Create(context.Background(), imp))
	require.Nil(s.T(), cl[0].WaitForImportCondition(imp, v1alpha1.ImportServiceCreated, true))

	require.Nil(s.T(), cl[0].Cluster().Resources().Delete(context.Background(), imp))
	require.Nil(s.T(), cl[0].Cluster().WaitForDeletion(&v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      imp.Name,
			Namespace: imp.Namespace,
		},
	}))

	imp2 := &v1alpha1.Import{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "imp2",
			Namespace: cl[0].Namespace(),
		},
		Spec: v1alpha1.ImportSpec{
			Port:       80,
			TargetPort: 1234,
			Sources:    []v1alpha1.ImportSource{{}},
		},
	}
	require.Nil(s.T(), cl[0].Cluster().Resources().Create(context.Background(), imp2))
	require.Nil(s.T(), cl[0].WaitForImportCondition(imp2, v1alpha1.ImportServiceCreated, true))
}
