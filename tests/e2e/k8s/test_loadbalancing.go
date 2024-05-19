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

package k8s

import (
	"context"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	"github.com/clusterlink-net/clusterlink/tests/e2e/k8s/services/httpecho"
	"github.com/clusterlink-net/clusterlink/tests/e2e/k8s/util"
)

func (s *TestSuite) TestLoadBalancingRoundRobin() {
	cl, err := s.fabric.DeployClusterlinks(3, nil)
	require.Nil(s.T(), err)

	importedService := &util.Service{
		Name: "my-import",
		Port: 80,
	}

	imp := &v1alpha1.Import{
		ObjectMeta: metav1.ObjectMeta{
			Name:      importedService.Name,
			Namespace: cl[0].Namespace(),
		},
		Spec: v1alpha1.ImportSpec{
			Port: importedService.Port,
		},
	}

	for i := 0; i < 3; i++ {
		require.Nil(s.T(), cl[0].CreatePeer(cl[i]))

		require.Nil(s.T(), cl[i].CreateService(&httpEchoService))
		require.Nil(s.T(), cl[i].CreateExport(&httpEchoService))
		require.Nil(s.T(), cl[i].CreatePolicy(util.PolicyAllowAll))

		imp.Spec.Sources = append(imp.Spec.Sources, v1alpha1.ImportSource{
			Peer:            cl[i].Name(),
			ExportName:      httpEchoService.Name,
			ExportNamespace: cl[i].Namespace(),
		})
	}

	require.Nil(s.T(), cl[0].Cluster().Resources().Create(context.Background(), imp))

	// test default lb scheme (round-robin)
	for i := 0; i < 30; i++ {
		data, err := cl[0].AccessService(httpecho.GetEchoValue, importedService, true, nil)
		require.Nil(s.T(), err)
		require.Equal(s.T(), data, cl[(i+1)%3].Name())
	}

	// take down first source
	require.Nil(s.T(), cl[0].DeleteExport(httpEchoService.Name))
	for i := 0; i < 30; i++ {
		data, err := cl[0].AccessService(httpecho.GetEchoValue, importedService, false, nil)
		require.Nil(s.T(), err)
		require.Equal(s.T(), data, cl[1+(i%2)].Name())
	}

	// take down second source
	require.Nil(s.T(), cl[1].DeleteExport(httpEchoService.Name))
	for i := 0; i < 30; i++ {
		data, err := cl[0].AccessService(httpecho.GetEchoValue, importedService, false, nil)
		require.Nil(s.T(), err)
		require.Equal(s.T(), data, cl[2].Name())
	}
}

func (s *TestSuite) TestLoadBalancingStatic() {
	cl, err := s.fabric.DeployClusterlinks(3, nil)
	require.Nil(s.T(), err)

	importedService := &util.Service{
		Name: "my-import",
		Port: 80,
	}

	imp := &v1alpha1.Import{
		ObjectMeta: metav1.ObjectMeta{
			Name:      importedService.Name,
			Namespace: cl[0].Namespace(),
		},
		Spec: v1alpha1.ImportSpec{
			LBScheme: v1alpha1.LBSchemeStatic,
			Port:     importedService.Port,
		},
	}

	for i := 0; i < 3; i++ {
		require.Nil(s.T(), cl[0].CreatePeer(cl[i]))

		require.Nil(s.T(), cl[i].CreateService(&httpEchoService))
		require.Nil(s.T(), cl[i].CreateExport(&httpEchoService))
		require.Nil(s.T(), cl[i].CreatePolicy(util.PolicyAllowAll))

		imp.Spec.Sources = append(imp.Spec.Sources, v1alpha1.ImportSource{
			Peer:            cl[i].Name(),
			ExportName:      httpEchoService.Name,
			ExportNamespace: cl[i].Namespace(),
		})
	}

	require.Nil(s.T(), cl[0].Cluster().Resources().Create(context.Background(), imp))

	// test static lb scheme
	for i := 0; i < 30; i++ {
		data, err := cl[0].AccessService(httpecho.GetEchoValue, importedService, true, nil)
		require.Nil(s.T(), err)
		require.Equal(s.T(), data, cl[0].Name())
	}

	// take down first source
	require.Nil(s.T(), cl[0].DeleteExport(httpEchoService.Name))
	for i := 0; i < 30; i++ {
		data, err := cl[0].AccessService(httpecho.GetEchoValue, importedService, false, nil)
		require.Nil(s.T(), err)
		require.Equal(s.T(), data, cl[1].Name())
	}

	// take down second source
	require.Nil(s.T(), cl[1].DeleteExport(httpEchoService.Name))
	for i := 0; i < 30; i++ {
		data, err := cl[0].AccessService(httpecho.GetEchoValue, importedService, false, nil)
		require.Nil(s.T(), err)
		require.Equal(s.T(), data, cl[2].Name())
	}
}

func (s *TestSuite) TestLoadBalancingRandom() {
	cl, err := s.fabric.DeployClusterlinks(3, nil)
	require.Nil(s.T(), err)

	importedService := &util.Service{
		Name: "my-import",
		Port: 80,
	}

	imp := &v1alpha1.Import{
		ObjectMeta: metav1.ObjectMeta{
			Name:      importedService.Name,
			Namespace: cl[0].Namespace(),
		},
		Spec: v1alpha1.ImportSpec{
			LBScheme: v1alpha1.LBSchemeRandom,
			Port:     importedService.Port,
		},
	}

	for i := 0; i < 3; i++ {
		require.Nil(s.T(), cl[0].CreatePeer(cl[i]))

		require.Nil(s.T(), cl[i].CreateService(&httpEchoService))
		require.Nil(s.T(), cl[i].CreateExport(&httpEchoService))
		require.Nil(s.T(), cl[i].CreatePolicy(util.PolicyAllowAll))

		imp.Spec.Sources = append(imp.Spec.Sources, v1alpha1.ImportSource{
			Peer:            cl[i].Name(),
			ExportName:      httpEchoService.Name,
			ExportNamespace: cl[i].Namespace(),
		})
	}

	require.Nil(s.T(), cl[0].Cluster().Resources().Create(context.Background(), imp))

	// test random lb scheme
	names := make(map[string]interface{})
	for i := 0; i < 100; i++ {
		data, err := cl[0].AccessService(httpecho.GetEchoValue, importedService, true, nil)
		require.Nil(s.T(), err)
		names[data] = nil
	}
	require.Equal(s.T(), 3, len(names))

	// take down first source
	require.Nil(s.T(), cl[0].DeleteExport(httpEchoService.Name))
	names = make(map[string]interface{})
	for i := 0; i < 30; i++ {
		data, err := cl[0].AccessService(httpecho.GetEchoValue, importedService, false, nil)
		require.Nil(s.T(), err)
		names[data] = nil
	}
	require.Equal(s.T(), 2, len(names))

	// take down second source
	require.Nil(s.T(), cl[1].DeleteExport(httpEchoService.Name))
	for i := 0; i < 30; i++ {
		data, err := cl[0].AccessService(httpecho.GetEchoValue, importedService, false, nil)
		require.Nil(s.T(), err)
		require.Equal(s.T(), data, cl[2].Name())
	}
}
