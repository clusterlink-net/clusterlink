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

package connectivitypdp_test

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	"github.com/clusterlink-net/clusterlink/pkg/policyengine/connectivitypdp"
)

const (
	testDir = "test_data"

	defaultNS = "default"
)

var (
	trivialLabel       = connectivitypdp.WorkloadAttrs{"key": "val"}
	trivialSelector    = metav1.LabelSelector{MatchLabels: trivialLabel}
	trivialWorkloadSet = v1alpha1.WorkloadSetOrSelector{WorkloadSelector: &trivialSelector}
)

func TestPrivilegedVsRegular(t *testing.T) {
	workloadSet := []v1alpha1.WorkloadSetOrSelector{trivialWorkloadSet}
	trivialConnPol := v1alpha1.AccessPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "reg",
			Namespace: defaultNS,
		},
		Spec: v1alpha1.AccessPolicySpec{
			Action: v1alpha1.AccessPolicyActionAllow,
			From:   workloadSet,
			To:     workloadSet,
		},
	}
	trivialPrivConnPol := v1alpha1.PrivilegedAccessPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "priv"},
		Spec: v1alpha1.AccessPolicySpec{
			Action: v1alpha1.AccessPolicyActionDeny,
			From:   workloadSet,
			To:     workloadSet,
		},
	}

	pdp := connectivitypdp.NewPDP()
	dests := []connectivitypdp.WorkloadAttrs{trivialLabel}
	decisions, err := pdp.Decide(trivialLabel, dests, defaultNS)
	require.Nil(t, err)
	require.Equal(t, connectivitypdp.DecisionDeny, decisions[0].Decision) // default deny
	require.Equal(t, connectivitypdp.DefaultDenyPolicyName, decisions[0].MatchedBy)
	require.Equal(t, false, decisions[0].PrivilegedMatch)

	err = pdp.AddOrUpdatePolicy(&trivialConnPol)
	require.Nil(t, err)
	dests = []connectivitypdp.WorkloadAttrs{trivialLabel}
	decisions, err = pdp.Decide(trivialLabel, dests, defaultNS)
	require.Nil(t, err)
	require.Equal(t, connectivitypdp.DecisionAllow, decisions[0].Decision) // regular allow policy allows connection
	require.Equal(t, types.NamespacedName{Name: "reg", Namespace: defaultNS}.String(), decisions[0].MatchedBy)
	require.Equal(t, false, decisions[0].PrivilegedMatch)

	err = pdp.AddOrUpdatePrivilegedPolicy(&trivialPrivConnPol)
	require.Nil(t, err)
	dests = []connectivitypdp.WorkloadAttrs{trivialLabel}
	decisions, err = pdp.Decide(trivialLabel, dests, defaultNS)
	require.Nil(t, err)
	require.Equal(t, connectivitypdp.DecisionDeny, decisions[0].Decision) // privileged deny policy denies connection
	require.Equal(t, types.NamespacedName{Name: "priv"}.String(), decisions[0].MatchedBy)
	require.Equal(t, true, decisions[0].PrivilegedMatch)
}

// TestAllLayers starts with one policy per layer (allow/deny X privileged/non-privileged)
// Policies are set s.t., they capture more connections as their priority is lower.
// We then test connections that should match the policy in a specific layer,
// but not policies in higher-priority layers.
// Finally we delete policies, starting with highest priority and going to lower priority policies.
// After each deletion we test again a specific connection, which should match all policies.
func TestAllLayers(t *testing.T) {
	pdp := connectivitypdp.NewPDP()
	err := addPoliciesFromFile(pdp, fileInTestDir("all_layers.yaml"))
	require.Nil(t, err)

	nonMeteringLabel := connectivitypdp.WorkloadAttrs{"workloadName": "non-metering-service"}
	meteringLabel := connectivitypdp.WorkloadAttrs{"workloadName": "global-metering-service"}
	privateMeteringLabel := connectivitypdp.WorkloadAttrs{"workloadName": "global-metering-service", "environment": "prod"}
	dests := []connectivitypdp.WorkloadAttrs{trivialLabel, nonMeteringLabel, meteringLabel, privateMeteringLabel}
	decisions, err := pdp.Decide(trivialLabel, dests, defaultNS)
	require.Nil(t, err)
	require.Equal(t, connectivitypdp.DecisionDeny, decisions[0].Decision) // default deny
	require.Equal(t, connectivitypdp.DefaultDenyPolicyName, decisions[0].MatchedBy)
	require.Equal(t, false, decisions[0].PrivilegedMatch)
	require.Equal(t, trivialLabel, decisions[0].Destination)
	require.Equal(t, connectivitypdp.DecisionAllow, decisions[1].Decision) // regular allow
	require.Equal(t, false, decisions[1].PrivilegedMatch)
	require.Equal(t, connectivitypdp.DecisionDeny, decisions[2].Decision) // regular deny
	require.Equal(t, false, decisions[2].PrivilegedMatch)
	require.Equal(t, connectivitypdp.DecisionAllow, decisions[3].Decision) // privileged allow
	require.Equal(t, true, decisions[3].PrivilegedMatch)

	privateLabel := map[string]string{"classification": "private", "environment": "prod"}
	dests = []connectivitypdp.WorkloadAttrs{privateMeteringLabel}
	decisions, err = pdp.Decide(privateLabel, dests, defaultNS)
	require.Nil(t, err)
	require.Equal(t, connectivitypdp.DecisionDeny, decisions[0].Decision) // privileged deny
	require.Equal(t, true, decisions[0].PrivilegedMatch)

	privDenyPolicy := getNameOfFirstPolicyInPDP(pdp, v1alpha1.AccessPolicyActionDeny, true)
	require.NotEmpty(t, privDenyPolicy)
	err = pdp.DeletePolicy(types.NamespacedName{Name: privDenyPolicy}, true)
	require.Nil(t, err)
	dests = []connectivitypdp.WorkloadAttrs{privateMeteringLabel}
	decisions, err = pdp.Decide(privateLabel, dests, defaultNS)
	require.Nil(t, err)
	// no privileged deny, so privileged allow matches
	require.Equal(t, connectivitypdp.DecisionAllow, decisions[0].Decision)

	privAllowPolicy := getNameOfFirstPolicyInPDP(pdp, v1alpha1.AccessPolicyActionAllow, true)
	require.NotEmpty(t, privAllowPolicy)
	err = pdp.DeletePolicy(types.NamespacedName{Name: privAllowPolicy}, true)
	require.Nil(t, err)
	dests = []connectivitypdp.WorkloadAttrs{privateMeteringLabel}
	decisions, err = pdp.Decide(privateLabel, dests, defaultNS)
	require.Nil(t, err)
	require.Equal(t, connectivitypdp.DecisionDeny, decisions[0].Decision) // no privileged allow, so regular deny matches

	regDenyPolicy := getNameOfFirstPolicyInPDP(pdp, v1alpha1.AccessPolicyActionDeny, false)
	require.NotEmpty(t, regDenyPolicy)
	err = pdp.DeletePolicy(types.NamespacedName{Name: regDenyPolicy, Namespace: defaultNS}, false)
	require.Nil(t, err)
	dests = []connectivitypdp.WorkloadAttrs{privateMeteringLabel}
	decisions, err = pdp.Decide(privateLabel, dests, defaultNS)
	require.Nil(t, err)
	require.Equal(t, connectivitypdp.DecisionAllow, decisions[0].Decision) // no regular deny, so regular allow matches

	regAllowPolicy := getNameOfFirstPolicyInPDP(pdp, v1alpha1.AccessPolicyActionAllow, false)
	require.NotEmpty(t, regAllowPolicy)
	err = pdp.DeletePolicy(types.NamespacedName{Name: regAllowPolicy, Namespace: defaultNS}, false)
	require.Nil(t, err)
	dests = []connectivitypdp.WorkloadAttrs{privateMeteringLabel}
	decisions, err = pdp.Decide(privateLabel, dests, defaultNS)
	require.Nil(t, err)
	require.Equal(t, connectivitypdp.DecisionDeny, decisions[0].Decision) // no regular allow, so default deny matches
}

func getNameOfFirstPolicyInPDP(pdp *connectivitypdp.PDP, action v1alpha1.AccessPolicyAction, privileged bool) string {
	if privileged {
		policies := pdp.GetPrivilegedPolicies()
		for idx := range policies {
			pol := &policies[idx]
			if pol.Spec.Action == action {
				return pol.Name
			}
		}
	} else {
		policies := pdp.GetPolicies()
		for idx := range policies {
			pol := &policies[idx]
			if pol.Spec.Action == action {
				return pol.Name
			}
		}
	}
	return ""
}

func TestDeleteNonexistingPolicies(t *testing.T) {
	pdp := connectivitypdp.NewPDP()
	err := pdp.DeletePolicy(types.NamespacedName{Name: "no-such-policy"}, true)
	require.NotNil(t, err)
	err = pdp.DeletePolicy(types.NamespacedName{Name: "no-such-policy"}, false)
	require.NotNil(t, err)
}

func TestBadSelector(t *testing.T) {
	badSelector := metav1.LabelSelector{MatchLabels: map[string]string{"this is not a key": "This val is bad!@#$%^"}}
	badWorkloadSet := v1alpha1.WorkloadSetOrSelector{WorkloadSelector: &badSelector}
	badSelectorPol := v1alpha1.AccessPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "aBadPolicy",
		},
		Spec: v1alpha1.AccessPolicySpec{
			Action: v1alpha1.AccessPolicyActionAllow,
			From:   []v1alpha1.WorkloadSetOrSelector{badWorkloadSet},
			To:     []v1alpha1.WorkloadSetOrSelector{trivialWorkloadSet},
		},
	}
	pdp := connectivitypdp.NewPDP()
	err := pdp.AddOrUpdatePolicy(&badSelectorPol)
	require.NotNil(t, err)
}

func TestNonexistingPolicyFile(t *testing.T) {
	pdp := connectivitypdp.NewPDP()
	err := addPoliciesFromFile(pdp, "no-such-file.yaml")
	require.NotNil(t, err)
}

func TestMalformedPolicyFile(t *testing.T) {
	pdp := connectivitypdp.NewPDP()
	err := addPoliciesFromFile(pdp, fileInTestDir("not_a_yaml"))
	require.NotNil(t, err)
}

func fileInTestDir(filename string) string {
	return filepath.Join(testDir, filename)
}

// addPoliciesFromFile takes a filename and reads all AccessPolicies
//
//	from this file.
//
// An error is returned if the file cannot be opened for reading.
// The file is expected to be a YAML/JSON file. Malformed files will return an error,
// but the file may contain manifests of other resources.
func addPoliciesFromFile(pdp *connectivitypdp.PDP, filename string) error {
	fileBuf, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed reading from file %s: %w", filename, err)
	}

	const lookaheadBufferSize = 200
	stringReader := strings.NewReader(string(fileBuf))
	decoder := yaml.NewYAMLOrJSONDecoder(stringReader, lookaheadBufferSize)
	for {
		var policy v1alpha1.AccessPolicy
		if err := decoder.Decode(&policy); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}

			return err
		}

		if policy.Kind == "PrivilegedAccessPolicy" {
			privPolicy := v1alpha1.PrivilegedAccessPolicy{
				ObjectMeta: metav1.ObjectMeta{Name: policy.Name},
				Spec:       policy.Spec,
			}
			err = pdp.AddOrUpdatePrivilegedPolicy(&privPolicy)
			if err != nil {
				fmt.Printf("invalid connectivity policy: %v\n", err)
			}
		} else {
			err = pdp.AddOrUpdatePolicy(&policy)
			if err != nil {
				fmt.Printf("invalid connectivity policy: %v\n", err)
			}
		}
	}
}
