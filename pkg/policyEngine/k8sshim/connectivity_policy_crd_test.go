package k8sshim_test

import (
	_ "embed"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.ibm.com/mbg-agent/pkg/policyEngine/k8sshim"
	"github.ibm.com/mbg-agent/pkg/policyEngine/policytypes"
)

//go:embed test_data/simple_privileged.yaml
var yamlBuffer string

func TestDecodeFromYaml(t *testing.T) {
	trivialLabel := map[string]string{"key": "val"}
	stringReader := strings.NewReader(yamlBuffer)
	var policy k8sshim.PrivilegedConnectivityPolicy
	err := yaml.NewYAMLOrJSONDecoder(stringReader, 200).Decode(&policy)
	require.Nil(t, err)
	require.Len(t, policy.Spec.ConnectionAttrs, 1)
	require.Equal(t, int32(5051), *policy.Spec.ConnectionAttrs[0].Port)
	matches, err := policy.ToInternal().Matches(trivialLabel, trivialLabel)
	require.Nil(t, err)
	require.False(t, matches)
	matchingLabel := policytypes.WorkloadAttrs{"workloadName": "global-metering-service"}
	matches, err = policy.ToInternal().Matches(trivialLabel, matchingLabel)
	require.Nil(t, err)
	require.True(t, matches)
}
