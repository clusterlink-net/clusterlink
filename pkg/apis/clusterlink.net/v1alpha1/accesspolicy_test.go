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

package v1alpha1_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
)

var (
	trivialLabel       = map[string]string{"key": "val"}
	trivialSelector    = metav1.LabelSelector{MatchLabels: trivialLabel}
	trivialWorkloadSet = v1alpha1.WorkloadSetOrSelector{WorkloadSelector: &trivialSelector}
)

func TestValidation(t *testing.T) {
	badPolicy := v1alpha1.AccessPolicy{}
	err := badPolicy.Validate()
	require.NotNil(t, err) // action is an empty string

	badPolicy.Spec.Action = "notAnAction"
	err = badPolicy.Validate()
	require.NotNil(t, err) // action is not a legal action

	badPolicy.Spec.Action = "deny"
	err = badPolicy.Validate()
	require.NotNil(t, err) // missing from

	badPolicy.Spec.From = []v1alpha1.WorkloadSetOrSelector{trivialWorkloadSet}
	err = badPolicy.Validate()
	require.NotNil(t, err) // missing to

	badPolicy.Spec.To = []v1alpha1.WorkloadSetOrSelector{trivialWorkloadSet}
	err = badPolicy.Validate()
	require.Nil(t, err)
}
