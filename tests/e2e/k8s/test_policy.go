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
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/authz"
	"github.com/clusterlink-net/clusterlink/tests/e2e/k8s/services"
	"github.com/clusterlink-net/clusterlink/tests/e2e/k8s/services/httpecho"
	"github.com/clusterlink-net/clusterlink/tests/e2e/k8s/util"
)

func (s *TestSuite) TestPolicyLabels() {
	cl, importedService := s.createTwoClustersWithEchoSvc()

	// 1. Create a policy that allows traffic only to the echo service at cl[0] - apply in cl[1] (on egress)
	//    In addition, create a policy to only allow traffic from cl[1] - apply in cl[0] (on ingress)
	allowEchoPolicyName := "allow-access-to-echo-svc"
	srcLabels := map[string]string{ // allow traffic only from cl1
		authz.PeerNameLabel: cl[1].Name(),
	}
	dstLabels := map[string]string{ // allow traffic only to echo in cl1
		authz.ServiceNameLabel:            httpEchoService.Name,
		authz.PeerNameLabel:               cl[0].Name(),
		authz.ServiceLabelsPrefix + "env": "test",
	}
	allowEchoPolicy := util.NewPolicy(allowEchoPolicyName, v1alpha1.AccessPolicyActionAllow, srcLabels, dstLabels)
	require.Nil(s.T(), cl[1].CreatePolicy(allowEchoPolicy))

	srcLabels = map[string]string{authz.PeerNameLabel: cl[1].Name()} // allow traffic only from cl1
	dstLabels = map[string]string{authz.PeerNameLabel: cl[0].Name()} // allow traffic only to cl0
	specificSrcPeerPolicy := util.NewPolicy("specific-peer", v1alpha1.AccessPolicyActionAllow, srcLabels, dstLabels)
	require.Nil(s.T(), cl[0].CreatePolicy(specificSrcPeerPolicy))

	data, err := cl[1].AccessService(httpecho.RunClientInPod, importedService, false, nil)
	require.Nil(s.T(), err)
	require.Equal(s.T(), cl[0].Name(), data)

	// 2. Add a "deny echo service" policy in cl[1] - should have a higher priority and so block the connection
	denyEchoPolicyName := "deny-access-to-echo"
	dstLabels = map[string]string{authz.ServiceNameLabel: httpEchoService.Name}
	denyEchoPolicy := util.NewPolicy(denyEchoPolicyName, v1alpha1.AccessPolicyActionDeny, nil, dstLabels)
	require.Nil(s.T(), cl[1].CreatePolicy(denyEchoPolicy))

	_, err = cl[1].AccessService(httpecho.RunClientInPod, importedService, false, nil)
	require.ErrorIs(s.T(), err, &util.PodFailedError{})

	// 3. Delete deny policy - connection is now allowed again
	require.Nil(s.T(), cl[1].DeletePolicy(denyEchoPolicyName))

	data, err = cl[1].AccessService(httpecho.RunClientInPod, importedService, false, nil)
	require.Nil(s.T(), err)
	require.Equal(s.T(), cl[0].Name(), data)

	// 4. Add a "deny peer cl0" policy in cl[1] - should have a higher priority and so block the connection
	denyCl0PolicyName := "deny-access-to-cl0"
	dstLabels = map[string]string{authz.PeerNameLabel: cl[0].Name()}
	denyCl0Policy := util.NewPolicy(denyCl0PolicyName, v1alpha1.AccessPolicyActionDeny, nil, dstLabels)
	require.Nil(s.T(), cl[1].CreatePolicy(denyCl0Policy))

	_, err = cl[1].AccessService(httpecho.RunClientInPod, importedService, false, nil)
	require.ErrorIs(s.T(), err, &util.PodFailedError{})

	// 5. Delete deny policy - connection is now allowed again
	require.Nil(s.T(), cl[1].DeletePolicy(denyCl0PolicyName))

	data, err = cl[1].AccessService(httpecho.RunClientInPod, importedService, false, nil)
	require.Nil(s.T(), err)
	require.Equal(s.T(), cl[0].Name(), data)

	// 6. Add a deny policy in cl[0] - should have a higher priority and so block the connection
	denyCl1PolicyName := "deny-access-from-cl1"
	denyCl1Policy := util.NewPolicy(denyCl1PolicyName, v1alpha1.AccessPolicyActionDeny, srcLabels, nil)
	require.Nil(s.T(), cl[0].CreatePolicy(denyCl1Policy))

	_, err = cl[1].AccessService(httpecho.RunClientInPod, importedService, false, nil)
	require.ErrorIs(s.T(), err, &util.PodFailedError{})

	// 7. Delete deny policy in cl[0] - connection is now allowed again
	require.Nil(s.T(), cl[0].DeletePolicy(denyCl1PolicyName))

	data, err = cl[1].AccessService(httpecho.RunClientInPod, importedService, false, nil)
	require.Nil(s.T(), err)
	require.Equal(s.T(), cl[0].Name(), data)

	// 8. Replace the policy in cl[1] with a policy having a wrong service name - connection should be denied
	require.Nil(s.T(), cl[1].DeletePolicy(allowEchoPolicyName))

	attrsWithBadSvcName := map[string]string{
		authz.ServiceNameLabel: "bad-svc",
		authz.PeerNameLabel:    cl[0].Name(),
	}
	badSvcPolicy := util.NewPolicy("bad-svc", v1alpha1.AccessPolicyActionAllow, nil, attrsWithBadSvcName)
	require.Nil(s.T(), cl[1].CreatePolicy(badSvcPolicy))

	_, err = cl[1].AccessService(httpecho.RunClientInPod, importedService, false, nil)
	require.ErrorIs(s.T(), err, &util.PodFailedError{})

	// 9. Add an allow policy in cl[1], but with a wrong service label - connection should still be denied
	attrsWithBadSvcLabels := map[string]string{
		authz.ServiceLabelsPrefix + "env": "prod",
	}
	badLabelPolicy := util.NewPolicy("bad-label", v1alpha1.AccessPolicyActionAllow, nil, attrsWithBadSvcLabels)
	require.Nil(s.T(), cl[1].CreatePolicy(badLabelPolicy))

	_, err = cl[1].AccessService(httpecho.RunClientInPod, importedService, false, nil)
	require.ErrorIs(s.T(), err, &util.PodFailedError{})

	// 10. Add an allow policy in cl[1], now with the right service label - connection should be allowed
	attrsWithGoodSvcLabels := map[string]string{
		authz.ServiceLabelsPrefix + "env": "test",
	}
	GoodLabelPolicy := util.NewPolicy("good-label", v1alpha1.AccessPolicyActionAllow, nil, attrsWithGoodSvcLabels)
	require.Nil(s.T(), cl[1].CreatePolicy(GoodLabelPolicy))

	data, err = cl[1].AccessService(httpecho.RunClientInPod, importedService, false, nil)
	require.Nil(s.T(), err)
	require.Equal(s.T(), cl[0].Name(), data)
}

func (s *TestSuite) TestPodAttributes() {
	cl, importedService := s.createTwoClustersWithEchoSvc()
	require.Nil(s.T(), cl[0].CreatePolicy(util.PolicyAllowAll))
	require.Nil(s.T(), cl[1].CreatePolicy(util.PolicyAllowAll))

	// 1. Sanity - just test that a pod can connect to echo service
	data, err := cl[1].AccessService(httpecho.RunClientInPod, importedService, false, nil)
	require.Nil(s.T(), err)
	require.Equal(s.T(), cl[0].Name(), data)

	// 2. Denying clients with a service account which is different from the Pod's SA - connection should work
	srcLabels := map[string]string{
		authz.ClientSALabel: "non-default",
	}
	denyNonDefaultSA := util.NewPolicy("deny-non-default-sa", v1alpha1.AccessPolicyActionDeny, srcLabels, nil)
	require.Nil(s.T(), cl[0].CreatePolicy(denyNonDefaultSA))
	require.Nil(s.T(), cl[1].CreatePolicy(denyNonDefaultSA))
	_, err = cl[1].AccessService(httpecho.RunClientInPod, importedService, false, nil)
	require.Nil(s.T(), err)

	// 3. Egress only - denying clients with a SA which equals the Pod's SA - connection should fail
	srcLabels = map[string]string{
		authz.ClientSALabel: "default",
	}
	denyDefaultSA := util.NewPolicy("deny-default-sa", v1alpha1.AccessPolicyActionDeny, srcLabels, nil)
	require.Nil(s.T(), cl[1].CreatePolicy(denyDefaultSA))
	_, err = cl[1].AccessService(httpecho.RunClientInPod, importedService, false, nil)
	require.NotNil(s.T(), err)
	require.Nil(s.T(), cl[1].DeletePolicy(denyDefaultSA.Name)) // revert

	// 4. Ingress only - denying clients with a SA which equals the Pod's SA - connection should fail
	require.Nil(s.T(), cl[0].CreatePolicy(denyDefaultSA))
	_, err = cl[1].AccessService(httpecho.RunClientInPod, importedService, false, nil)
	require.NotNil(s.T(), err)
	require.Nil(s.T(), cl[0].DeletePolicy(denyDefaultSA.Name)) // revert

	// 5. Egress only - denying client-pods with a label different from the client pod - connection should work
	selRequirement := metav1.LabelSelectorRequirement{
		Key:      authz.ClientLabelsPrefix + "app",
		Operator: metav1.LabelSelectorOpNotIn,
		Values:   []string{httpecho.EchoClientPodName},
	}
	labelSelector := metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{selRequirement}}
	denyOthers := util.NewPolicyFromLabelSelectors("deny-others", v1alpha1.AccessPolicyActionDeny, &labelSelector, nil)
	require.Nil(s.T(), cl[1].CreatePolicy(denyOthers))
	_, err = cl[1].AccessService(httpecho.RunClientInPod, importedService, false, nil)
	require.Nil(s.T(), err)
}

func (s *TestSuite) TestPrivilegedPolicies() {
	cl, importedService := s.createTwoClustersWithEchoSvc()
	require.Nil(s.T(), cl[0].CreatePolicy(util.PolicyAllowAll))

	dstLabels := map[string]string{
		authz.ServiceNameLabel: httpEchoService.Name,
		authz.PeerNameLabel:    cl[0].Name(),
	}

	privDenyPolicyName := "priv-deny"
	privilegedDenyPolicy := util.NewPrivilegedPolicy(privDenyPolicyName, v1alpha1.AccessPolicyActionDeny, nil, dstLabels)
	require.Nil(s.T(), cl[1].CreatePrivilegedPolicy(privilegedDenyPolicy))

	privAllowPolicyName := "priv-allow"
	privilegedAllowPolicy := util.NewPrivilegedPolicy(privAllowPolicyName, v1alpha1.AccessPolicyActionAllow, nil, dstLabels)
	require.Nil(s.T(), cl[1].CreatePrivilegedPolicy(privilegedAllowPolicy))

	regDenyPolicyName := "reg-deny"
	regDenyPolicy := util.NewPolicy(regDenyPolicyName, v1alpha1.AccessPolicyActionDeny, nil, dstLabels)
	require.Nil(s.T(), cl[1].CreatePolicy(regDenyPolicy))

	regAllowPolicyName := "reg-allow"
	regAllowPolicy := util.NewPolicy(regAllowPolicyName, v1alpha1.AccessPolicyActionAllow, nil, dstLabels)
	require.Nil(s.T(), cl[1].CreatePolicy(regAllowPolicy))

	// 1. privileged deny has highest priority -> connection is denied
	_, err := cl[1].AccessService(httpecho.GetEchoValue, importedService, true, &services.ConnectionResetError{})
	require.ErrorIs(s.T(), err, &services.ConnectionResetError{})

	// 2. deleting privileged deny -> privileged allow now has highest priority -> connection is allowed
	require.Nil(s.T(), cl[1].DeletePrivilegedPolicy(privDenyPolicyName))

	data, err := cl[1].AccessService(httpecho.GetEchoValue, importedService, true, nil)
	require.Nil(s.T(), err)
	require.Equal(s.T(), cl[0].Name(), data)

	// 3. deleting privileged allow -> non-privileged deny now has highest priority -> connection is denied
	require.Nil(s.T(), cl[1].DeletePrivilegedPolicy(privAllowPolicyName))

	_, err = cl[1].AccessService(httpecho.GetEchoValue, importedService, true, &services.ConnectionResetError{})
	require.ErrorIs(s.T(), err, &services.ConnectionResetError{})

	// 4. deleting non-privileged deny -> non-privileged allow now has highest priority -> connection is allowed
	require.Nil(s.T(), cl[1].DeletePolicy(regDenyPolicyName))

	data, err = cl[1].AccessService(httpecho.GetEchoValue, importedService, true, nil)
	require.Nil(s.T(), err)
	require.Equal(s.T(), cl[0].Name(), data)

	// 5. deleting non-privileged allow -> default deny takes place -> connection is denied
	require.Nil(s.T(), cl[1].DeletePolicy(regAllowPolicyName))

	_, err = cl[1].AccessService(httpecho.GetEchoValue, importedService, true, &services.ConnectionResetError{})
	require.ErrorIs(s.T(), err, &services.ConnectionResetError{})
}

func (s *TestSuite) createTwoClustersWithEchoSvc() ([]*util.ClusterLink, *util.Service) {
	cl, err := s.fabric.DeployClusterlinks(2, nil)
	require.Nil(s.T(), err)

	require.Nil(s.T(), cl[0].CreateService(&httpEchoService))
	require.Nil(s.T(), cl[0].CreateExport(&httpEchoService))
	require.Nil(s.T(), cl[1].CreatePeer(cl[0]))

	importedService := &util.Service{
		Name:   httpEchoService.Name,
		Port:   80,
		Labels: httpEchoService.Labels,
	}
	require.Nil(s.T(), cl[1].CreateImport(importedService, cl[0], httpEchoService.Name))
	return cl, importedService
}
