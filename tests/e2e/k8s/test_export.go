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
	"fmt"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	"github.com/clusterlink-net/clusterlink/tests/e2e/k8s/services/httpecho"
	"github.com/clusterlink-net/clusterlink/tests/e2e/k8s/util"
)

func (s *TestSuite) TestExportHost() {
	cl, err := s.fabric.DeployClusterlinks(1, nil)
	require.Nil(s.T(), err)

	// create export using Spec.Host to point to a service from a different namespace
	export := &v1alpha1.Export{
		ObjectMeta: metav1.ObjectMeta{
			Name:      httpEchoService.Name,
			Namespace: cl[0].Namespace(),
		},
		Spec: v1alpha1.ExportSpec{
			Host: fmt.Sprintf("%s.%s.svc.cluster.local", httpEchoService.Name, httpEchoService.Namespace),
			Port: httpEchoService.Port,
		},
	}
	require.Nil(s.T(), cl[0].Cluster().Resources().Create(context.Background(), export))

	// create an import using the above export
	importedService := &util.Service{
		Name: "import",
		Port: 80,
	}
	require.Nil(s.T(), cl[0].CreateImport(importedService, cl[0], export.Name))

	// create peer and policy to allow import access
	require.Nil(s.T(), cl[0].CreatePolicy(util.PolicyAllowAll))
	require.Nil(s.T(), cl[0].CreatePeer(cl[0]))

	// check export status is valid
	require.Nil(s.T(), cl[0].WaitForExportCondition(export, v1alpha1.ExportValid, true))

	// check access for imported service
	data, err := cl[0].AccessService(httpecho.GetEchoValue, importedService, true, nil)
	require.Nil(s.T(), err)
	require.Equal(s.T(), cl[0].Name(), data)
}

func (s *TestSuite) TestExportServiceNotExist() {
	cl, err := s.fabric.DeployClusterlinks(1, nil)
	require.Nil(s.T(), err)

	// create an export that points to a non-existing service
	export := &v1alpha1.Export{
		ObjectMeta: metav1.ObjectMeta{
			Name:      httpEchoService.Name,
			Namespace: cl[0].Namespace(),
		},
		Spec: v1alpha1.ExportSpec{
			Port: httpEchoService.Port,
		},
	}
	require.Nil(s.T(), cl[0].Cluster().Resources().Create(context.Background(), export))

	// create an import of the above export
	importedService := &util.Service{
		Name: "import",
		Port: 80,
	}
	require.Nil(s.T(), cl[0].CreateImport(importedService, cl[0], export.Name))

	// create peer and policy to allow import access
	require.Nil(s.T(), cl[0].CreatePolicy(util.PolicyAllowAll))
	require.Nil(s.T(), cl[0].CreatePeer(cl[0]))

	// verify export status indicates invalid
	require.Nil(s.T(), cl[0].WaitForExportCondition(export, v1alpha1.ExportValid, false))

	// create the service the export refers to
	require.Nil(s.T(), cl[0].CreateService(&httpEchoService))

	// wait for the export status to change to valid
	require.Nil(s.T(), cl[0].WaitForExportCondition(export, v1alpha1.ExportValid, true))

	// verify access to imported service
	data, err := cl[0].AccessService(httpecho.GetEchoValue, importedService, true, nil)
	require.Nil(s.T(), err)
	require.Equal(s.T(), cl[0].Name(), data)

	// delete the service used by the export
	require.Nil(s.T(), cl[0].DeleteService(httpEchoService.Name))
	// verify that the export status again indicates invalid
	require.Nil(s.T(), cl[0].WaitForExportCondition(export, v1alpha1.ExportValid, false))
}
