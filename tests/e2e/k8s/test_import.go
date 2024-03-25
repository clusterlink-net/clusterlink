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

	// create an import with an explicit target port
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

	// verify import service is created
	require.Nil(s.T(), cl[0].WaitForImportCondition(imp1, v1alpha1.ImportServiceCreated, true))

	// create a second import with the same explicit target port
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

	// verify import status indicates a conflict
	require.Nil(s.T(), cl[0].WaitForImportCondition(imp2, v1alpha1.ImportServiceCreated, false))
	require.True(s.T(), meta.IsStatusConditionFalse(imp2.Status.Conditions, v1alpha1.ImportTargetPortValid))

	// verify that service for the second import was not created
	imp2Service := &util.Service{
		Name: imp2.Name,
		Port: imp2.Spec.Port,
	}
	_, err = cl[0].AccessService(httpecho.GetEchoValue, imp2Service, true, &services.ServiceNotFoundError{})
	require.ErrorIs(s.T(), err, &services.ServiceNotFoundError{})

	// update the target port of the second import to some other free port
	imp2.Spec.TargetPort = 1235
	require.Nil(s.T(), cl[0].Cluster().Resources().Update(context.Background(), imp2))

	// verify the status is now good
	require.Nil(s.T(), cl[0].WaitForImportCondition(imp2, v1alpha1.ImportServiceCreated, true))
	require.True(s.T(), meta.IsStatusConditionTrue(imp2.Status.Conditions, v1alpha1.ImportTargetPortValid))

	// second import service should now exist (but return RST as it has no sources)
	_, err = cl[0].AccessService(httpecho.GetEchoValue, imp2Service, true, &services.ConnectionResetError{})
	require.ErrorIs(s.T(), err, &services.ConnectionResetError{})
}

func (s *TestSuite) TestImportConflictingService() {
	cl, err := s.fabric.DeployClusterlinks(1, nil)
	require.Nil(s.T(), err)

	// create a service
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

	// create an import matching the previously created service
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

	// verify import status indicates service could not be created (due to conflict)
	require.Nil(s.T(), cl[0].WaitForImportCondition(imp, v1alpha1.ImportServiceCreated, false))

	// update the service to look as service managed by clusterlink
	service.Labels = make(map[string]string)
	service.Labels[control.LabelManagedBy] = control.AppName
	service.Labels[control.LabelImportName] = imp.Name
	service.Labels[control.LabelImportNamespace] = imp.Namespace
	require.Nil(s.T(), cl[0].Cluster().Resources().Update(context.Background(), service))

	// check import status reflects that service was updated successfully
	require.Nil(s.T(), cl[0].WaitForImportCondition(imp, v1alpha1.ImportServiceCreated, true))

	// verify service exist (RST instead of service not found)
	impService := &util.Service{
		Name: imp.Name,
		Port: imp.Spec.Port,
	}
	_, err = cl[0].AccessService(httpecho.GetEchoValue, impService, true, &services.ConnectionResetError{})
	require.ErrorIs(s.T(), err, &services.ConnectionResetError{})

	// delete import service
	require.Nil(s.T(), cl[0].DeleteService(service.Name))

	// import service should be re-created, and we should eventually get RST instead of service not found
	_, err = cl[0].AccessService(httpecho.GetEchoValue, impService, true, &services.ConnectionResetError{})
	require.ErrorIs(s.T(), err, &services.ConnectionResetError{})
	// verify status is good
	require.Nil(s.T(), cl[0].WaitForImportCondition(imp, v1alpha1.ImportServiceCreated, true))

	// update service managed-by label to non-clusterlink
	require.Nil(s.T(), cl[0].Cluster().Resources().Get(context.Background(), service.Name, service.Namespace, service))
	service.Labels[control.LabelManagedBy] = "other"
	require.Nil(s.T(), cl[0].Cluster().Resources().Update(context.Background(), service))

	// verify import status indicates invalid
	require.Nil(s.T(), cl[0].WaitForImportCondition(imp, v1alpha1.ImportServiceCreated, false))

	// update managed-by label back to clusterlink
	service.Labels[control.LabelManagedBy] = control.AppName
	require.Nil(s.T(), cl[0].Cluster().Resources().Update(context.Background(), service))

	// verify access and status are back ok
	require.Nil(s.T(), cl[0].WaitForImportCondition(imp, v1alpha1.ImportServiceCreated, true))
	_, err = cl[0].AccessService(httpecho.GetEchoValue, impService, true, &services.ConnectionResetError{})
	require.ErrorIs(s.T(), err, &services.ConnectionResetError{})
}

func (s *TestSuite) TestImportUnprivilegedNamespace() {
	cl, err := s.fabric.DeployClusterlinks(1, nil)
	require.Nil(s.T(), err)

	// create a user-namespace for creating end-user imports (not in clusterlink-system)
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

	// create a service to conflict with the import system service
	require.Nil(s.T(), cl[0].Cluster().Resources().Create(context.Background(), systemService))

	// create an import whose system service conflicts with the previously created service
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

	// verify status indicates invalid
	require.Nil(s.T(), cl[0].WaitForImportCondition(imp, v1alpha1.ImportServiceCreated, false))

	// delete the conflicting system service
	require.Nil(s.T(), cl[0].Cluster().Resources().Delete(context.Background(), systemService))
	// wait for status to be good
	require.Nil(s.T(), cl[0].WaitForImportCondition(imp, v1alpha1.ImportServiceCreated, true))

	// update the import service to a bad service managed by other
	require.Nil(s.T(), cl[0].Cluster().Resources().Get(context.Background(), service.Name, service.Namespace, service))
	service.Labels[control.LabelManagedBy] = "other"
	service.Spec.ExternalName = "broken"
	require.Nil(s.T(), cl[0].Cluster().Resources().Update(context.Background(), service))

	// wait for status to indicate invalid
	require.Nil(s.T(), cl[0].WaitForImportCondition(imp, v1alpha1.ImportServiceCreated, false))

	// return service to be managed by clusterlink
	service.Labels[control.LabelManagedBy] = control.AppName
	require.Nil(s.T(), cl[0].Cluster().Resources().Update(context.Background(), service))

	// wait for status to be good
	require.Nil(s.T(), cl[0].WaitForImportCondition(imp, v1alpha1.ImportServiceCreated, true))

	// verify imported service exists (getting a RST since no sources are defined)
	impService := &util.Service{
		Name:      imp.Name,
		Namespace: namespace,
		Port:      imp.Spec.Port,
	}
	_, err = cl[0].AccessService(httpecho.GetEchoValue, impService, true, &services.ConnectionResetError{})
	require.ErrorIs(s.T(), err, &services.ConnectionResetError{})

	// delete import
	require.Nil(s.T(), cl[0].Cluster().Resources().Delete(context.Background(), imp))
	// wait for both services to be deleted
	require.Nil(s.T(), cl[0].Cluster().WaitForDeletion(service))
	require.Nil(s.T(), cl[0].Cluster().WaitForDeletion(systemService))
}

func (s *TestSuite) TestImportDelete() {
	cl, err := s.fabric.DeployClusterlinks(1, nil)
	require.Nil(s.T(), err)

	// create import with an explicit target port
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

	// wait for status to indicate service was created
	require.Nil(s.T(), cl[0].WaitForImportCondition(imp, v1alpha1.ImportServiceCreated, true))

	// delete import
	require.Nil(s.T(), cl[0].Cluster().Resources().Delete(context.Background(), imp))
	// wait for import service to be deleted
	require.Nil(s.T(), cl[0].Cluster().WaitForDeletion(&v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      imp.Name,
			Namespace: imp.Namespace,
		},
	}))

	// create a second import with the same target port (which should be back free)
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
	// verify status is good
	require.Nil(s.T(), cl[0].WaitForImportCondition(imp2, v1alpha1.ImportServiceCreated, true))
}
