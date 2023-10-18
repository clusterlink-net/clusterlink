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

package k8sshim_test

import (
	_ "embed"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/clusterlink-net/clusterlink/pkg/policyengine/k8sshim"
	"github.com/clusterlink-net/clusterlink/pkg/policyengine/policytypes"
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
