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
	"encoding/json"
	"fmt"

	"github.com/stretchr/testify/require"

	"github.com/clusterlink-net/clusterlink/pkg/api"
	"github.com/clusterlink-net/clusterlink/pkg/policyengine"
	"github.com/clusterlink-net/clusterlink/pkg/policyengine/policytypes"
	"github.com/clusterlink-net/clusterlink/tests/e2e/k8s/services"
	"github.com/clusterlink-net/clusterlink/tests/e2e/k8s/services/httpecho"
	"github.com/clusterlink-net/clusterlink/tests/e2e/k8s/util"
)

func (s *TestSuite) TestConnectivity() {
	s.RunOnAllDataplaneTypes(func(cfg *util.PeerConfig) {
		cl, err := s.fabric.DeployClusterlinks(2, cfg)
		require.Nil(s.T(), err)

		require.Nil(s.T(), cl[0].CreateExport("echo", &httpEchoService))
		require.Nil(s.T(), cl[0].CreatePolicy(util.PolicyAllowAll))
		require.Nil(s.T(), cl[1].CreatePeer(cl[0]))

		importedService := &util.Service{
			Name: "echo",
			Port: 80,
		}
		require.Nil(s.T(), cl[1].CreateImport("echo", importedService))

		require.Nil(s.T(), cl[1].CreateBinding("echo", cl[0]))
		require.Nil(s.T(), cl[1].CreatePolicy(util.PolicyAllowAll))

		data, err := cl[1].AccessService(httpecho.GetEchoValue, importedService, true, nil)
		require.Nil(s.T(), err)
		require.Equal(s.T(), cl[0].Name(), data)
	})
}

func (s *TestSuite) TestControlplaneCRUD() {
	s.RunOnAllDataplaneTypes(func(cfg *util.PeerConfig) {
		cfg.ControlplanePersistency = true
		cl, err := s.fabric.DeployClusterlinks(3, cfg)
		require.Nil(s.T(), err)

		client0 := cl[0].Client()
		client1 := cl[1].Client()

		// test import API
		imp := api.Import{
			Name: "echo",
			Spec: api.ImportSpec{
				Service: api.Endpoint{
					Host: "echo",
					Port: 1234,
				},
			},
		}

		// list imports when empty
		objects, err := client0.Imports.List()
		require.Nil(s.T(), err)
		require.Empty(s.T(), objects.(*[]api.Import))

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
			Name: imp.Spec.Service.Host,
			Port: imp.Spec.Service.Port,
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
		importFromServer := *objects.(*api.Import)
		require.Equal(s.T(), importFromServer.Name, imp.Name)
		require.Equal(s.T(), importFromServer.Spec, imp.Spec)
		require.Equal(s.T(), importFromServer.Status.Listener.Host, "")
		require.NotZero(s.T(), importFromServer.Status.Listener.Port)

		// list imports
		objects, err = client0.Imports.List()
		require.Nil(s.T(), err)
		require.ElementsMatch(s.T(), *objects.(*[]api.Import), []api.Import{importFromServer})

		// test binding API
		binding := api.Binding{
			Spec: api.BindingSpec{
				Import: imp.Name,
				Peer:   cl[1].Name(),
			},
		}

		// list bindings when empty
		objects, err = client0.Bindings.List()
		require.Nil(s.T(), err)
		require.Empty(s.T(), objects.(*[]api.Binding))

		// get non-existing binding
		_, err = client0.Bindings.Get(binding.Spec.Import)
		require.Nil(s.T(), err)
		require.Empty(s.T(), objects.(*[]api.Binding))

		// delete non-existing binding
		require.NotNil(s.T(), client0.Bindings.Delete(&binding))
		// update non-existing binding
		require.NotNil(s.T(), client0.Bindings.Update(&binding))
		// create binding
		require.Nil(s.T(), client0.Bindings.Create(&binding))
		// create binding which already exists
		require.NotNil(s.T(), client0.Bindings.Create(&binding))
		// update binding
		require.Nil(s.T(), client0.Bindings.Update(&binding))

		// verify no access
		_, err = accessService(false, &services.ConnectionResetError{})
		require.ErrorIs(s.T(), err, &services.ConnectionResetError{})

		// add another binding (for testing binding get vs list)
		binding2 := api.Binding{
			Spec: api.BindingSpec{
				Import: "dummy",
				Peer:   "dummy",
			},
		}
		require.Nil(s.T(), client0.Bindings.Create(&binding2))

		// get bindings
		objects, err = client0.Bindings.Get(binding.Spec.Import)
		require.Nil(s.T(), err)
		require.ElementsMatch(s.T(), *objects.(*[]api.Binding), []api.Binding{binding})

		// list bindings
		objects, err = client0.Bindings.List()
		require.Nil(s.T(), err)
		require.ElementsMatch(s.T(), *objects.(*[]api.Binding), []api.Binding{binding, binding2})

		// test peer API
		peer := api.Peer{
			Name: cl[1].Name(),
			Spec: api.PeerSpec{
				Gateways: []api.Endpoint{{
					Host: cl[1].IP(),
					Port: cl[1].Port(),
				}},
			},
		}

		// list peers when empty
		objects, err = client0.Peers.List()
		require.Nil(s.T(), err)
		require.Empty(s.T(), objects.(*[]api.Peer))

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
		peerFromServer := *objects.(*api.Peer)
		require.Equal(s.T(), peerFromServer.Name, peer.Name)
		require.Equal(s.T(), peerFromServer.Spec, peer.Spec)
		require.Equal(s.T(), peerFromServer.Status, api.PeerStatus{
			State:    "",
			LastSeen: "",
		})

		// list peers
		objects, err = client0.Peers.List()
		require.Nil(s.T(), err)
		require.ElementsMatch(s.T(), *objects.(*[]api.Peer), []api.Peer{peerFromServer})

		// add another peer (for upcoming load-balancing test)
		peer2 := api.Peer{
			Name: cl[2].Name(),
			Spec: api.PeerSpec{
				Gateways: []api.Endpoint{{
					Host: cl[2].IP(),
					Port: cl[2].Port(),
				}},
			},
		}
		require.Nil(s.T(), client0.Peers.Create(&peer2))

		data, err := json.Marshal(util.PolicyAllowAll)
		require.Nil(s.T(), err)

		// test access policy API
		policy := api.Policy{
			Name: "allow-all",
			Spec: api.PolicySpec{
				Blob: data,
			},
		}

		// list access policies when empty
		objects, err = client0.AccessPolicies.List()
		require.Nil(s.T(), err)
		require.Empty(s.T(), objects.(*[]api.Policy))

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
		require.Equal(s.T(), *objects.(*api.Policy), policy)

		// list access policies
		objects, err = client0.AccessPolicies.List()
		require.Nil(s.T(), err)
		require.ElementsMatch(s.T(), *objects.(*[]api.Policy), []api.Policy{policy})

		// test export API
		export := api.Export{
			Name: imp.Name,
			Spec: api.ExportSpec{
				Service: api.Endpoint{
					Host: fmt.Sprintf(
						"%s.%s.svc.cluster.local",
						httpEchoService.Name, httpEchoService.Namespace),
					Port: httpEchoService.Port,
				},
			},
		}

		// list exports when empty
		objects, err = client1.Exports.List()
		require.Nil(s.T(), err)
		require.Empty(s.T(), objects.(*[]api.Export))

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
		require.Equal(s.T(), *objects.(*api.Export), export)

		// list exports
		objects, err = client1.Exports.List()
		require.Nil(s.T(), err)
		require.ElementsMatch(s.T(), *objects.(*[]api.Export), []api.Export{export})

		// allow export to be accessed
		require.Nil(s.T(), client1.AccessPolicies.Create(&policy))
		// verify access
		str, err := accessService(true, nil)
		require.Nil(s.T(), err)
		require.Equal(s.T(), str, cl[1].Name())

		// test LB policy API
		staticPolicy := &policyengine.LBPolicy{
			ServiceSrc:  policyengine.Wildcard,
			ServiceDst:  imp.Name,
			Scheme:      policyengine.Static,
			DefaultPeer: cl[1].Name(),
		}

		data, err = json.Marshal(staticPolicy)
		require.Nil(s.T(), err)

		lbPolicy := api.Policy{
			Name: "static",
			Spec: api.PolicySpec{
				Blob: data,
			},
		}

		// list LB policies when empty
		objects, err = client0.LBPolicies.List()
		require.Nil(s.T(), err)
		require.Empty(s.T(), objects.(*[]api.Policy))

		// get non-existing LB policy
		_, err = client0.LBPolicies.Get(lbPolicy.Name)
		require.NotNil(s.T(), err)

		// delete non-existing LB policy
		require.NotNil(s.T(), client0.LBPolicies.Delete(lbPolicy.Name))
		// update non-existing LB policy
		require.NotNil(s.T(), client0.LBPolicies.Update(&lbPolicy))
		// create LB policy
		require.Nil(s.T(), client0.LBPolicies.Create(&lbPolicy))
		// create LB policy which already exists
		require.NotNil(s.T(), client0.LBPolicies.Create(&lbPolicy))

		// create false binding to verify LB policy
		binding3 := api.Binding{
			Spec: api.BindingSpec{
				Import: imp.Name,
				Peer:   cl[2].Name(),
			},
		}
		require.Nil(s.T(), client0.Bindings.Create(&binding3))

		// verify access
		str, err = accessService(false, nil)
		require.Nil(s.T(), err)
		require.Equal(s.T(), str, cl[1].Name())

		// get LB policy
		objects, err = client0.LBPolicies.Get(lbPolicy.Name)
		require.Nil(s.T(), err)
		require.Equal(s.T(), *objects.(*api.Policy), lbPolicy)

		// list LB policies
		objects, err = client0.LBPolicies.List()
		require.Nil(s.T(), err)
		require.ElementsMatch(s.T(), *objects.(*[]api.Policy), []api.Policy{lbPolicy})

		// TODO: currently broken
		// // update import port
		// imp.Spec.Service.Port++
		// require.Nil(s.T(), client0.Imports.Update(&imp))
		// // verify no access to previous port
		// _, err = accessService(true, &services.ConnectionRefusedError{})
		// require.ErrorIs(s.T(), err, &services.ConnectionRefusedError{})
		// // verify access to new port
		// importedService.Port++
		// _, err = accessService(true, nil)
		// require.Nil(s.T(), err)
		//
		// // update import host
		// imp.Spec.Service.Host += "2"
		// require.Nil(s.T(), client0.Imports.Update(&imp))
		// // verify no access to previous host
		// _, err = accessService(true, &services.ServiceNotFoundError{})
		// require.ErrorIs(s.T(), err, &services.ServiceNotFoundError{})
		// // verify access to new host
		// importedService.Name += "2"
		// _, err = accessService(true, nil)
		// require.Nil(s.T(), err)
		// // get import after update
		// objects, err = client0.Imports.Get(imp.Name)
		// require.Nil(s.T(), err)
		// require.Equal(s.T(), objects.(*api.Import).Spec, imp.Spec)
		// require.Equal(s.T(), objects.(*api.Import).Status, importFromServer.Status)
		// importFromServer = *objects.(*api.Import)

		// update peer
		peer.Spec.Gateways[0].Port++
		require.Nil(s.T(), client0.Peers.Update(&peer))
		// get peer after update
		objects, err = client0.Peers.Get(peer.Name)
		require.Nil(s.T(), err)
		require.Equal(s.T(), objects.(*api.Peer).Spec, peer.Spec)
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
		policy2.Action = policytypes.ActionDeny
		data, err = json.Marshal(&policy2)
		require.Nil(s.T(), err)
		oldPolicyBlob := policy.Spec.Blob
		policy.Spec.Blob = data
		require.Nil(s.T(), client0.AccessPolicies.Update(&policy))
		// get access policy after update
		objects, err = client0.AccessPolicies.Get(policy.Name)
		require.Nil(s.T(), err)
		require.Equal(s.T(), objects.(*api.Policy).Spec, policy.Spec)
		// verify no access after update
		_, err = accessService(false, &services.ConnectionResetError{})
		require.ErrorIs(s.T(), err, &services.ConnectionResetError{})
		//  update access policy back
		policy.Spec.Blob = oldPolicyBlob
		require.Nil(s.T(), client0.AccessPolicies.Update(&policy))
		// verify access after update back
		str, err = accessService(false, nil)
		require.Nil(s.T(), err)
		require.Equal(s.T(), str, cl[1].Name())

		// update export
		export.Spec.Service.Port++
		require.Nil(s.T(), client1.Exports.Update(&export))
		// get export after update
		objects, err = client1.Exports.Get(export.Name)
		require.Nil(s.T(), err)
		require.Equal(s.T(), objects.(*api.Export).Spec, export.Spec)
		// verify no access after update
		_, err = accessService(true, &services.ConnectionResetError{})
		require.ErrorIs(s.T(), err, &services.ConnectionResetError{})
		// update export back
		export.Spec.Service.Port--
		require.Nil(s.T(), client1.Exports.Update(&export))
		// verify access after update back
		str, err = accessService(true, nil)
		require.Nil(s.T(), err)
		require.Equal(s.T(), str, cl[1].Name())

		// update LB policy
		staticPolicy.DefaultPeer = cl[2].Name()
		data, err = json.Marshal(staticPolicy)
		require.Nil(s.T(), err)
		oldPolicyBlob = lbPolicy.Spec.Blob
		lbPolicy.Spec.Blob = data
		require.Nil(s.T(), client0.LBPolicies.Update(&lbPolicy))
		// get LB policy after update
		objects, err = client0.LBPolicies.Get(lbPolicy.Name)
		require.Nil(s.T(), err)
		require.Equal(s.T(), objects.(*api.Policy).Spec, lbPolicy.Spec)
		// verify no access after update
		_, err = accessService(false, &services.ConnectionResetError{})
		require.ErrorIs(s.T(), err, &services.ConnectionResetError{})
		// update LB policy back
		lbPolicy.Spec.Blob = oldPolicyBlob
		require.Nil(s.T(), client0.LBPolicies.Update(&lbPolicy))
		// verify access after update back
		str, err = accessService(false, nil)
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
		importFromServer = *objects.(*api.Import)
		// verify access after re-create
		str, err = accessService(true, nil)
		require.Nil(s.T(), err)
		require.Equal(s.T(), str, cl[1].Name())

		// TODO: currently broken
		// // delete binding
		// require.Nil(s.T(), client0.Bindings.Delete(&binding))
		// // get binding after delete
		// objects, err = client0.Bindings.Get(imp.Name)
		// require.Nil(s.T(), err)
		// require.ElementsMatch(s.T(), *objects.(*[]api.Binding), []api.Binding{binding3})
		// // verify no access after delete
		// _, err = accessService(false, &services.ConnectionResetError{})
		// require.ErrorIs(s.T(), err, &services.ConnectionResetError{})
		// // re-create binding
		// require.Nil(s.T(), client0.Bindings.Create(&binding))
		// // verify access after re-create
		// str, err = accessService(false, nil)
		// require.Nil(s.T(), err)
		// require.Equal(s.T(), str, cl[1].Name())

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

		// delete LB policy
		require.Nil(s.T(), client0.LBPolicies.Delete(lbPolicy.Name))
		// get LB policy after delete
		_, err = client0.LBPolicies.Get(lbPolicy.Name)
		require.NotNil(s.T(), err)
		// verify random access after delete
		_, err = accessService(true, &services.ConnectionResetError{})
		require.ErrorIs(s.T(), err, &services.ConnectionResetError{})
		// re-create LB policy
		require.Nil(s.T(), client0.LBPolicies.Create(&lbPolicy))
		// verify access after re-create
		str, err = accessService(false, nil)
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
		require.ElementsMatch(s.T(), *objects.(*[]api.Import), []api.Import{importFromServer})

		// verify bindings after restart
		objects, err = client0.Bindings.List()
		require.Nil(s.T(), err)
		require.ElementsMatch(s.T(), *objects.(*[]api.Binding), []api.Binding{binding, binding2, binding3})

		// verify peers after restart
		objects, err = client0.Peers.List()
		require.Nil(s.T(), err)
		require.ElementsMatch(s.T(), *objects.(*[]api.Peer), []api.Peer{peerFromServer, peer2})

		// verify access policies after restart
		objects, err = client0.AccessPolicies.List()
		require.Nil(s.T(), err)
		require.ElementsMatch(s.T(), *objects.(*[]api.Policy), []api.Policy{policy})

		// verify exports after restart
		objects, err = client1.Exports.List()
		require.Nil(s.T(), err)
		require.ElementsMatch(s.T(), *objects.(*[]api.Export), []api.Export{export})

		// verify lb policies after restart
		objects, err = client0.LBPolicies.List()
		require.Nil(s.T(), err)
		require.ElementsMatch(s.T(), *objects.(*[]api.Policy), []api.Policy{lbPolicy})

		// verify access after restart
		str, err = accessService(true, nil)
		require.Nil(s.T(), err)
		require.Equal(s.T(), str, cl[1].Name())
	})
}
