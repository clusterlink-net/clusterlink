package connectivitypdp_test

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.ibm.com/mbg-agent/pkg/policyEngine/connectivitypdp"
	"github.ibm.com/mbg-agent/pkg/policyEngine/k8sshim"
	"github.ibm.com/mbg-agent/pkg/policyEngine/policytypes"
)

const testDir = "test_data"

var (
	trivialLabel       = map[string]string{"key": "val"}
	trivialSelector    = metav1.LabelSelector{MatchLabels: trivialLabel}
	trivialWorkloadSet = policytypes.WorkloadSetOrSelector{WorkloadSelector: &trivialSelector}
	emptyPrivPolicy    = policytypes.ConnectivityPolicy{Privileged: true}
	emptyRegPolicy     = policytypes.ConnectivityPolicy{Privileged: false}
)

func TestPrivilegedVsRegular(t *testing.T) {
	workloadSet := []policytypes.WorkloadSetOrSelector{trivialWorkloadSet}
	trivialConnPol := policytypes.ConnectivityPolicy{
		Name: "reg", Privileged: false, Action: policytypes.PolicyActionAllow,
		From: workloadSet, To: workloadSet}
	trivialPrivConnPol := policytypes.ConnectivityPolicy{
		Name: "reg", Privileged: true, Action: policytypes.PolicyActionDeny,
		From: workloadSet, To: workloadSet}

	pdp := connectivitypdp.NewPDP()
	res, err := pdp.Decide(trivialLabel, trivialLabel)
	require.Nil(t, err)
	require.Equal(t, policytypes.PolicyActionDeny, res) // default deny

	err = pdp.AddOrUpdatePolicy(trivialConnPol)
	require.Nil(t, err)
	res, err = pdp.Decide(trivialLabel, trivialLabel)
	require.Nil(t, err)
	require.Equal(t, policytypes.PolicyActionAllow, res) // regular allow policy allows connection

	err = pdp.AddOrUpdatePolicy(trivialPrivConnPol)
	require.Nil(t, err)
	res, err = pdp.Decide(trivialLabel, trivialLabel)
	require.Nil(t, err)
	require.Equal(t, policytypes.PolicyActionDeny, res) // privileged deny policy denies connection
}

// TestAllLayers starts with one policy per layer (allow/deny x privileged/non/privileged)
// Policies are set s.t., they capture more connections as their priority is lower.
// We then test connections that should match the policy in a specific layer, but not policies in higher-priority layers.
// Finally we delete policies, starting with highest priority and going to lower priority policies.
// After each deletion we test again a specific connection, which should match all policies.
func TestAllLayers(t *testing.T) {
	pdp := connectivitypdp.NewPDP()
	err := addPoliciesFromFile(pdp, fileInTestDir("all_layers.yaml"))
	require.Nil(t, err)

	res, err := pdp.Decide(trivialLabel, trivialLabel)
	require.Nil(t, err)
	require.Equal(t, policytypes.PolicyActionDeny, res) // default deny

	nonMeteringLabel := policytypes.WorkloadAttrs{"workloadName": "non-metering-service"}
	res, err = pdp.Decide(trivialLabel, nonMeteringLabel)
	require.Nil(t, err)
	require.Equal(t, policytypes.PolicyActionAllow, res) // regular allow

	meteringLabel := policytypes.WorkloadAttrs{"workloadName": "global-metering-service"}
	res, err = pdp.Decide(trivialLabel, meteringLabel)
	require.Nil(t, err)
	require.Equal(t, policytypes.PolicyActionDeny, res) // regular deny

	meteringLabel["environment"] = "prod"
	res, err = pdp.Decide(trivialLabel, meteringLabel)
	require.Nil(t, err)
	require.Equal(t, policytypes.PolicyActionAllow, res) // privileged allow

	privateLabel := map[string]string{"classification": "private", "environment": "prod"}
	res, err = pdp.Decide(privateLabel, meteringLabel)
	require.Nil(t, err)
	require.Equal(t, policytypes.PolicyActionDeny, res) // privileged deny

	privDenyPolicy := getNameOfFirstPolicyInPDP(pdp, policytypes.PolicyActionDeny, true)
	require.NotEmpty(t, privDenyPolicy)
	err = pdp.DeletePolicy(privDenyPolicy, true)
	require.Nil(t, err)
	res, err = pdp.Decide(privateLabel, meteringLabel)
	require.Nil(t, err)
	require.Equal(t, policytypes.PolicyActionAllow, res) // no privileged deny, so privileged allow matches

	privAllowPolicy := getNameOfFirstPolicyInPDP(pdp, policytypes.PolicyActionAllow, true)
	require.NotEmpty(t, privAllowPolicy)
	err = pdp.DeletePolicy(privAllowPolicy, true)
	require.Nil(t, err)
	res, err = pdp.Decide(privateLabel, meteringLabel)
	require.Nil(t, err)
	require.Equal(t, policytypes.PolicyActionDeny, res) // no privileged allow, so regular deny matches

	regDenyPolicy := getNameOfFirstPolicyInPDP(pdp, policytypes.PolicyActionDeny, false)
	require.NotEmpty(t, regDenyPolicy)
	err = pdp.DeletePolicy(regDenyPolicy, false)
	require.Nil(t, err)
	res, err = pdp.Decide(privateLabel, meteringLabel)
	require.Nil(t, err)
	require.Equal(t, policytypes.PolicyActionAllow, res) // no regular deny, so regular allow matches

	regAllowPolicy := getNameOfFirstPolicyInPDP(pdp, policytypes.PolicyActionAllow, false)
	require.NotEmpty(t, regAllowPolicy)
	err = pdp.DeletePolicy(regAllowPolicy, false)
	require.Nil(t, err)
	res, err = pdp.Decide(privateLabel, meteringLabel)
	require.Nil(t, err)
	require.Equal(t, policytypes.PolicyActionDeny, res) // no regular allow, so default deny matches
}

func getNameOfFirstPolicyInPDP(pdp *connectivitypdp.PDP, action policytypes.PolicyAction, privileged bool) string {
	policies := pdp.GetPolicies()
	for _, pol := range policies {
		if pol.Action == action && pol.Privileged == privileged {
			return pol.Name
		}
	}
	return ""
}

func TestDeleteNonexistingPolicies(t *testing.T) {
	pdp := connectivitypdp.NewPDP()
	err := pdp.DeletePolicy("no-such-policy", true)
	require.NotNil(t, err)
	err = pdp.DeletePolicy("no-such-policy", false)
	require.NotNil(t, err)
}

func TestBadSelector(t *testing.T) {
	badSelector := metav1.LabelSelector{MatchLabels: map[string]string{"this is not a key": "This val is bad!@#$%^"}}
	badWorkloadSet := policytypes.WorkloadSetOrSelector{WorkloadSelector: &badSelector}
	badSelectorPol := policytypes.ConnectivityPolicy{
		Name:   "aBadPolicy",
		Action: policytypes.PolicyActionAllow,
		From:   []policytypes.WorkloadSetOrSelector{badWorkloadSet},
		To:     []policytypes.WorkloadSetOrSelector{trivialWorkloadSet}}
	pdp := connectivitypdp.NewPDP()
	err := pdp.AddOrUpdatePolicy(badSelectorPol)
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

// addPoliciesFromFile takes a filename and reads all PrivilegedConnectivityPolicies
// as well as all ConnectivityPolicies from this file.
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
		var policy k8sshim.PrivilegedConnectivityPolicy
		err := decoder.Decode(&policy)
		switch err {
		case nil:
			switch policy.Kind {
			case k8sshim.PrivilegedConnectivityPolicyKind:
				err = pdp.AddOrUpdatePolicy(*policy.ToInternal())
				if err != nil {
					fmt.Printf("invalid privileged connectivity policy: %v\n", err)
				}
			case k8sshim.ConnectivityPolicyKind:
				regPolicy := k8sshim.ConnectivityPolicy{ObjectMeta: metav1.ObjectMeta{Name: policy.Name}, Spec: policy.Spec}
				err = pdp.AddOrUpdatePolicy(*regPolicy.ToInternal())
				if err != nil {
					fmt.Printf("invalid connectivity policy: %v\n", err)
				}
			default: // TODO: log a warning
				fmt.Printf("Object kind is not a connectivity policy: %s\n", policy.Kind)
			}
		case io.EOF:
			return nil
		default:
			return err
		}
	}
}
