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

package util

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
)

var PolicyAllowAll = &v1alpha1.AccessPolicy{
	ObjectMeta: metav1.ObjectMeta{
		Name: "allow-all",
	},
	Spec: v1alpha1.AccessPolicySpec{
		Action: v1alpha1.AccessPolicyActionAllow,
		From: v1alpha1.WorkloadSetOrSelectorList{{
			WorkloadSelector: &metav1.LabelSelector{},
		}},
		To: v1alpha1.WorkloadSetOrSelectorList{{
			WorkloadSelector: &metav1.LabelSelector{},
		}},
	},
}
