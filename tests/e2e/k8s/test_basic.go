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
	"fmt"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	"github.com/clusterlink-net/clusterlink/tests/e2e/k8s/services"
	"github.com/clusterlink-net/clusterlink/tests/e2e/k8s/services/httpecho"
	"github.com/clusterlink-net/clusterlink/tests/e2e/k8s/util"
)

func (s *TestSuite) TestConnectivity() {
	s.RunOnAllDataplaneTypes(func(cfg *util.PeerConfig) {
		cl, err := s.fabric.DeployClusterlinks(2, cfg)
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
	})
}

func (s *TestSuite) TestControlplaneCRUD() {
	s.RunOnAllDataplaneTypes(func(cfg *util.PeerConfig) {
		cfg.ControlplanePersistency = true
		cfg.CRUDMode = true
		cl, err := s.fabric.DeployClusterlinks(3, cfg)
		require.Nil(s.T(), err)

		client0 := cl[0].Client()
		client1 := cl[1].Client()

		// test import API
		imp := v1alpha1.Import{
			ObjectMeta: metav1.ObjectMeta{
				Name: httpEchoService.Name,
			},
			Spec: v1alpha1.ImportSpec{
				Port:    1234,
				Sources: []v1alpha1.ImportSource{{Peer: cl[1].Name(), ExportName: httpEchoService.Name, ExportNamespace: cl[1].Namespace()}},
			},
		}

		// list imports when empty
		objects, err := client0.Imports.List()
		require.Nil(s.T(), err)
		require.Empty(s.T(), objects.(*[]v1alpha1.Import))

		// get non-existing import
		_, err = client0.Imports.Get(imp.Name)
		require.NotNil(s.T(), err)

		// delete non-existing import
		require.NotNil(s.T(), client0.Imports.Delete(imp.Name))
		// update non-existing import
		require.NotNil(s.T(), client0.Imports.Update(&imp))
		// create import
		require.Nil(s.T(), client0.Imports.Create(&imp))
		// create import when it already exists
		require.NotNil(s.T(), client0.Imports.Create(&imp))

		importedService := &util.Service{
			Name: imp.Name,
			Port: imp.Spec.Port,
		}

		accessService := func(allowRetry bool, expectedError error) (string, error) {
			return cl[0].AccessService(
				httpecho.GetEchoValue, importedService, allowRetry, expectedError)
		}

		// verify import listener is up
		_, err = accessService(true, &services.ConnectionResetError{})
		require.ErrorIs(s.T(), err, &services.ConnectionResetError{})

		// get import
		objects, err = client0.Imports.Get(imp.Name)
		require.Nil(s.T(), err)
		importFromServer := *objects.(*v1alpha1.Import)
		require.Equal(s.T(), importFromServer.Name, imp.Name)
		require.Equal(s.T(), importFromServer.Spec.Port, imp.Spec.Port)
		require.Equal(s.T(), importFromServer.Spec.Sources, imp.Spec.Sources)
		require.NotZero(s.T(), importFromServer.Spec.TargetPort)

		// list imports
		objects, err = client0.Imports.List()
		require.Nil(s.T(), err)
		require.ElementsMatch(s.T(), *objects.(*[]v1alpha1.Import), []v1alpha1.Import{importFromServer})

		// test peer API
		peer := v1alpha1.Peer{
			ObjectMeta: metav1.ObjectMeta{
				Name: cl[1].Name(),
			},
			Spec: v1alpha1.PeerSpec{
				Gateways: []v1alpha1.Endpoint{{
					Host: cl[1].IP(),
					Port: cl[1].Port(),
				}},
			},
		}

		// list peers when empty
		objects, err = client0.Peers.List()
		require.Nil(s.T(), err)
		require.Empty(s.T(), objects.(*[]v1alpha1.Peer))

		// get non-existing peer
		_, err = client0.Peers.Get(peer.Name)
		require.NotNil(s.T(), err)

		// delete non-existing peer
		require.NotNil(s.T(), client0.Peers.Delete(peer.Name))
		// update non-existing peer
		require.NotNil(s.T(), client0.Peers.Update(&peer))
		// create peer
		require.Nil(s.T(), client0.Peers.Create(&peer))
		// create peer which already exists
		require.NotNil(s.T(), client0.Peers.Create(&peer))

		// verify no access
		_, err = accessService(false, &services.ConnectionResetError{})
		require.ErrorIs(s.T(), err, &services.ConnectionResetError{})

		// get peer
		objects, err = client0.Peers.Get(peer.Name)
		require.Nil(s.T(), err)
		peerFromServer := *objects.(*v1alpha1.Peer)
		require.Equal(s.T(), peerFromServer.Name, peer.Name)
		require.Equal(s.T(), peerFromServer.Spec, peer.Spec)

		// list peers
		objects, err = client0.Peers.List()
		require.Nil(s.T(), err)
		if !assert.ElementsMatch(s.T(), *objects.(*[]v1alpha1.Peer), []v1alpha1.Peer{peerFromServer}) {
			objects, err = client0.Peers.Get(peer.Name)
			require.Nil(s.T(), err)
			peerFromServer = *objects.(*v1alpha1.Peer)
		}
		require.ElementsMatch(s.T(), *objects.(*[]v1alpha1.Peer), []v1alpha1.Peer{peerFromServer})

		// add another peer (for upcoming load-balancing test)
		peer2 := v1alpha1.Peer{
			ObjectMeta: metav1.ObjectMeta{
				Name: cl[2].Name(),
			},
			Spec: v1alpha1.PeerSpec{
				Gateways: []v1alpha1.Endpoint{{
					Host: cl[2].IP(),
					Port: cl[2].Port(),
				}},
			},
		}
		require.Nil(s.T(), client0.Peers.Create(&peer2))

		// test access policy API
		policy := *util.PolicyAllowAll

		// list access policies when empty
		objects, err = client0.AccessPolicies.List()
		require.Nil(s.T(), err)
		require.Empty(s.T(), objects.(*[]v1alpha1.AccessPolicy))

		// get non-existing access policy
		_, err = client0.AccessPolicies.Get(policy.Name)
		require.NotNil(s.T(), err)

		// delete non-existing access policy
		require.NotNil(s.T(), client0.AccessPolicies.Delete(policy.Name))
		// update non-existing access policy
		require.NotNil(s.T(), client0.AccessPolicies.Update(&policy))
		// create access policy
		require.Nil(s.T(), client0.AccessPolicies.Create(&policy))
		// create access policy which already exists
		require.NotNil(s.T(), client0.AccessPolicies.Create(&policy))

		// verify no access
		_, err = accessService(false, &services.ConnectionResetError{})
		require.ErrorIs(s.T(), err, &services.ConnectionResetError{})

		// get access policy
		objects, err = client0.AccessPolicies.Get(policy.Name)
		require.Nil(s.T(), err)
		require.Equal(s.T(), *objects.(*v1alpha1.AccessPolicy), policy)

		// list access policies
		objects, err = client0.AccessPolicies.List()
		require.Nil(s.T(), err)
		require.ElementsMatch(s.T(), *objects.(*[]v1alpha1.AccessPolicy), []v1alpha1.AccessPolicy{policy})

		// test export API
		export := v1alpha1.Export{
			ObjectMeta: metav1.ObjectMeta{
				Name: imp.Name,
			},
			Spec: v1alpha1.ExportSpec{
				Host: fmt.Sprintf(
					"%s.%s.svc.cluster.local",
					httpEchoService.Name, httpEchoService.Namespace),
				Port: httpEchoService.Port,
			},
		}

		// list exports when empty
		objects, err = client1.Exports.List()
		require.Nil(s.T(), err)
		require.Empty(s.T(), objects.(*[]v1alpha1.Export))

		// get non-existing export
		_, err = client1.Exports.Get(export.Name)
		require.NotNil(s.T(), err)

		// delete non-existing export
		require.NotNil(s.T(), client1.Exports.Delete(export.Name))
		// update non-existing export
		require.NotNil(s.T(), client1.Exports.Update(&export))
		// create export
		require.Nil(s.T(), client1.Exports.Create(&export))
		// create export which already exists
		require.NotNil(s.T(), client1.Exports.Create(&export))

		// verify no access
		_, err = accessService(false, &services.ConnectionResetError{})
		require.ErrorIs(s.T(), err, &services.ConnectionResetError{})

		// get export
		objects, err = client1.Exports.Get(export.Name)
		require.Nil(s.T(), err)
		exportFromServer := *objects.(*v1alpha1.Export)
		require.Equal(s.T(), export.Name, exportFromServer.Name)
		require.Equal(s.T(), export.Spec, exportFromServer.Spec)

		// list exports
		objects, err = client1.Exports.List()
		require.Nil(s.T(), err)
		require.ElementsMatch(s.T(), *objects.(*[]v1alpha1.Export), []v1alpha1.Export{exportFromServer})

		// allow export to be accessed
		require.Nil(s.T(), client1.AccessPolicies.Create(&policy))
		// verify access
		str, err := accessService(true, nil)
		require.Nil(s.T(), err)
		require.Equal(s.T(), str, cl[1].Name())

		// create false binding to verify LB policy
		imp.Spec.Sources = append(imp.Spec.Sources,
			v1alpha1.ImportSource{Peer: cl[2].Name(), ExportName: httpEchoService.Name, ExportNamespace: cl[2].Namespace()})
		imp.Spec.LBScheme = v1alpha1.LBSchemeStatic
		require.Nil(s.T(), client0.Imports.Update(&imp))

		// verify access
		str, err = accessService(false, nil)
		require.Nil(s.T(), err)
		require.Equal(s.T(), str, cl[1].Name())

		// update import port
		imp.Spec.Port++
		require.Nil(s.T(), client0.Imports.Update(&imp))
		// verify no access to previous port
		_, err = accessService(true, &services.ConnectionRefusedError{})
		require.ErrorIs(s.T(), err, &services.ConnectionRefusedError{})
		// verify access to new port
		importedService.Port++
		_, err = accessService(true, nil)
		require.Nil(s.T(), err)

		// update peer
		peer.Spec.Gateways[0].Port++
		require.Nil(s.T(), client0.Peers.Update(&peer))
		// get peer after update
		objects, err = client0.Peers.Get(peer.Name)
		require.Nil(s.T(), err)
		require.Equal(s.T(), objects.(*v1alpha1.Peer).Spec, peer.Spec)
		// verify no access after update
		_, err = accessService(true, &services.ConnectionResetError{})
		require.ErrorIs(s.T(), err, &services.ConnectionResetError{})
		// update peer back
		peer.Spec.Gateways[0].Port--
		require.Nil(s.T(), client0.Peers.Update(&peer))
		// verify access after update back
		str, err = accessService(true, nil)
		require.Nil(s.T(), err)
		require.Equal(s.T(), str, cl[1].Name())

		//  update access policy
		policy2 := *util.PolicyAllowAll
		policy2.Spec.Action = v1alpha1.AccessPolicyActionDeny
		require.Nil(s.T(), client0.AccessPolicies.Update(&policy2))
		// get access policy after update
		objects, err = client0.AccessPolicies.Get(policy2.Name)
		require.Nil(s.T(), err)
		require.Equal(s.T(), objects.(*v1alpha1.AccessPolicy).Spec, policy2.Spec)
		// verify no access after update
		_, err = accessService(false, &services.ConnectionResetError{})
		require.ErrorIs(s.T(), err, &services.ConnectionResetError{})
		//  update access policy back
		require.Nil(s.T(), client0.AccessPolicies.Update(&policy))
		// verify access after update back
		str, err = accessService(false, nil)
		require.Nil(s.T(), err)
		require.Equal(s.T(), str, cl[1].Name())

		// update export
		export.Spec.Port++
		require.Nil(s.T(), client1.Exports.Update(&export))
		// get export after update
		objects, err = client1.Exports.Get(export.Name)
		require.Nil(s.T(), err)
		require.Equal(s.T(), objects.(*v1alpha1.Export).Spec, export.Spec)
		// verify no access after update
		_, err = accessService(true, &services.ConnectionResetError{})
		require.ErrorIs(s.T(), err, &services.ConnectionResetError{})
		// update export back
		export.Spec.Port--
		require.Nil(s.T(), client1.Exports.Update(&export))
		// verify access after update back
		str, err = accessService(true, nil)
		require.Nil(s.T(), err)
		require.Equal(s.T(), str, cl[1].Name())

		// delete import
		require.Nil(s.T(), client0.Imports.Delete(imp.Name))
		// get import after delete
		_, err = client0.Imports.Get(imp.Name)
		require.NotNil(s.T(), err)
		// verify no access after delete
		_, err = accessService(true, &services.ServiceNotFoundError{})
		require.ErrorIs(s.T(), err, &services.ServiceNotFoundError{})
		// re-create import
		require.Nil(s.T(), client0.Imports.Create(&imp))
		// re-get import from server
		objects, err = client0.Imports.Get(imp.Name)
		require.Nil(s.T(), err)
		importFromServer = *objects.(*v1alpha1.Import)
		// verify access after re-create
		str, err = accessService(true, nil)
		require.Nil(s.T(), err)
		require.Equal(s.T(), str, cl[1].Name())

		// delete peer
		require.Nil(s.T(), client0.Peers.Delete(peer.Name))
		// get peer after delete
		_, err = client0.Peers.Get(peer.Name)
		require.NotNil(s.T(), err)
		// verify no access after delete
		_, err = accessService(true, &services.ConnectionResetError{})
		require.ErrorIs(s.T(), err, &services.ConnectionResetError{})
		// re-create peer
		require.Nil(s.T(), client0.Peers.Create(&peer))
		// verify access after re-create
		str, err = accessService(true, nil)
		require.Nil(s.T(), err)
		require.Equal(s.T(), str, cl[1].Name())

		// delete access policy
		require.Nil(s.T(), client0.AccessPolicies.Delete(policy.Name))
		// get access policy after delete
		_, err = client0.AccessPolicies.Get(policy.Name)
		require.NotNil(s.T(), err)
		// verify no access after delete
		_, err = accessService(false, &services.ConnectionResetError{})
		require.ErrorIs(s.T(), err, &services.ConnectionResetError{})
		// re-create access policy
		require.Nil(s.T(), client0.AccessPolicies.Create(&policy))
		// verify access after re-create
		str, err = accessService(false, nil)
		require.Nil(s.T(), err)
		require.Equal(s.T(), str, cl[1].Name())

		// delete export
		require.Nil(s.T(), client1.Exports.Delete(export.Name))
		// get export after delete
		_, err = client1.Exports.Get(export.Name)
		require.NotNil(s.T(), err)
		// verify no access after delete
		_, err = accessService(true, &services.ConnectionResetError{})
		require.ErrorIs(s.T(), err, &services.ConnectionResetError{})
		// re-create export
		require.Nil(s.T(), client1.Exports.Create(&export))
		// verify access after re-create
		str, err = accessService(true, nil)
		require.Nil(s.T(), err)
		require.Equal(s.T(), str, cl[1].Name())

		// restart controlplanes
		runner := util.AsyncRunner{}
		runner.Run(cl[0].RestartControlplane)
		runner.Run(cl[1].RestartControlplane)
		require.Nil(s.T(), runner.Wait())

		// verify imports after restart
		objects, err = client0.Imports.List()
		require.Nil(s.T(), err)
		require.ElementsMatch(s.T(), *objects.(*[]v1alpha1.Import), []v1alpha1.Import{importFromServer})

		// verify 2 peers after restart
		objects, err = client0.Peers.List()
		require.Nil(s.T(), err)
		require.Equal(s.T(), len(*objects.(*[]v1alpha1.Peer)), 2)

		// verify access policies after restart
		objects, err = client0.AccessPolicies.List()
		require.Nil(s.T(), err)
		require.ElementsMatch(s.T(), *objects.(*[]v1alpha1.AccessPolicy), []v1alpha1.AccessPolicy{policy})

		// verify exports after restart
		objects, err = client1.Exports.List()
		require.Nil(s.T(), err)
		require.Equal(s.T(), len(*objects.(*[]v1alpha1.Export)), 1)

		// verify access after restart
		str, err = accessService(true, nil)
		require.Nil(s.T(), err)
		require.Equal(s.T(), str, cl[1].Name())
	})
}
